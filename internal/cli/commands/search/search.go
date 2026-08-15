// Package search 提供影片/实体搜索命令与以图搜番图片模式。
package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	magnetspkg "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/magnets"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/image"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the movie/dimension search command and the image reverse search mode.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var (
		page, limit   int
		zone, sort    string
		filterBy, typ string
		hasMagnets    bool
		asJSON        bool
		asNDJSON      bool
		asImage       bool
		source        string
		noCache       bool
		magnets       int
		cnsub, hd     bool
		minSize       string
	)
	cmd := &cobra.Command{
		Use:   "search KEYWORD|IMAGE",
		Short: "Search movies (or other dimensions with --type)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := pipeline.ResolveOutputMode(asNDJSON, asJSON, streams.OutIsTerminal)
			if err != nil {
				return err
			}
			// --magnets 仅支持 movie 搜索。
			if magnets > 0 || cmd.Flags().Changed("magnets") {
				if typ != "" && typ != "movie" {
					return fmt.Errorf("--magnets is only supported for movie search (got --type %q)", typ)
				}
				if !cmd.Flags().Changed("cnsub") && !cmd.Flags().Changed("hd") && !cmd.Flags().Changed("min-size") && magnets == 0 {
					// --magnets 0 不配合筛选 flag 时仍允许（返回全部磁力）。
				}
			}
			reader := bufio.NewReader(streams.In)
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			imageMode, inputs, err := classifySearchInput(arg, asImage, reader, streams)
			if err != nil {
				return err
			}
			if imageMode {
				if cmd.Flags().Changed("magnets") {
					return runImageSearchMagnets(options, streams, arg, reader, source, noCache, mode, magnets, cnsub, hd, minSize)
				}
				return runImageSearch(options, streams, arg, reader, source, noCache, mode)
			}
			if len(inputs) == 0 {
				return fmt.Errorf("keyword or an image")
			}
			if cmd.Flags().Changed("magnets") {
				return runSearchMagnets(options, streams, inputs, mode, page, limit, zone, sort, filterBy, hasMagnets, magnets, cnsub, hd, minSize)
			}
			runner := &pipeline.BatchRunner{
				Name:       "search",
				LegacyJSON: true,
				// 非 TTY 单项也输出可供下游消费的稳定番号记录。
				RouteTextThroughPipeline: true,
				Kinds:                    []pipeline.Kind{pipeline.KindMovie},
				ClientFactory: func() (*javdb.Client, error) {
					return client.New(options, "")
				},
				RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
					return runSearchOne(c, ctx, input, page, limit, zone, sort, filterBy, typ, hasMagnets)
				},
				RunMany: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) ([]pipeline.Envelope, error) {
					return runSearchMany(c, ctx, input, page, limit, zone, sort, filterBy, typ, hasMagnets)
				},
				Legacy: func(args []string) error {
					return runTextSearch(options, streams, args[0], page, limit, zone, sort, filterBy, typ, hasMagnets, asJSON)
				},
			}
			return runner.ExecuteWithInputs(streams, inputs, mode)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (0 = server default)")
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2|all")
	cmd.Flags().StringVar(&sort, "sort", "", "relevance|release|score|update|hit")
	cmd.Flags().StringVar(&filterBy, "filter-by", "", "can_play|magnets|subtitle|single")
	cmd.Flags().StringVar(&typ, "type", "", "movie|code|series|actor|maker|director|list")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop movie rows with magnets_count == 0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	cmd.Flags().BoolVar(&asImage, "image", false, "Treat the argument as an image path or HTTP(S) URL")
	cmd.Flags().StringVar(&source, "source", "", "Reverse-search source (default: reverse_search.default_source)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass the reverse-search response cache")
	cmd.Flags().IntVar(&magnets, "magnets", 0, "Fetch magnets for each result: 0=all, N=top N by best rule")
	cmd.Flags().BoolVar(&cnsub, "cnsub", false, "Only magnets with Chinese subtitles (requires --magnets)")
	cmd.Flags().BoolVar(&hd, "hd", false, "Only HD magnets (requires --magnets)")
	cmd.Flags().StringVar(&minSize, "min-size", "", "Min magnet size e.g. 2000, 4GB, 500MB (requires --magnets)")
	return cmd
}

// classifySearchInput 一次性分类输入：返回图片模式标志与批处理输入。
//   - --image 强制图片；参数是现有普通文件或 HTTP(S) URL 自动图片。
//   - 无参数非 TTY stdin 按固定顺序分类（图片 magic → NDJSON → 文本）。
//   - 位置参数与非空 stdin 同时存在是歧义错误，绝不静默丢数据。
func classifySearchInput(arg string, forced bool, reader *bufio.Reader, streams *invocation.Streams) (bool, []pipeline.Envelope, error) {
	stdinHasContent := false
	if !streams.InIsTerminal {
		if _, err := reader.Peek(1); err == nil {
			stdinHasContent = true
		} else if !errors.Is(err, io.EOF) {
			return false, nil, fmt.Errorf("read stdin: %w", err)
		}
	}
	if arg != "" && stdinHasContent {
		return false, nil, fmt.Errorf("ambiguous input: provide either a positional keyword/image or stdin, not both")
	}
	if arg != "" {
		if forced || isHTTPURL(arg) || isRegularFile(arg) {
			return true, nil, nil
		}
		return false, []pipeline.Envelope{pipeline.New("", arg, "")}, nil
	}
	if streams.InIsTerminal || !stdinHasContent {
		return false, nil, nil
	}
	classification, content, err := pipeline.Classify(reader)
	if err != nil {
		return false, nil, err
	}
	switch classification {
	case pipeline.ClassificationImage:
		return true, nil, nil
	case pipeline.ClassificationNDJSON:
		inputs, err := pipeline.ParseBatch(content, pipeline.KindNDJSONInput)
		return false, inputs, err
	default:
		inputs, err := pipeline.ParseBatch(content, pipeline.KindMovie)
		return false, inputs, err
	}
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func isRegularFile(value string) bool {
	info, err := os.Stat(value)
	return err == nil && info.Mode().IsRegular()
}

// runSearchOne 执行单个关键词搜索并返回管道信封：movie 类型 → kind movie，
// 其他 --type → 对应实体 kind。
func runSearchOne(c *javdb.Client, ctx context.Context, input pipeline.Envelope, page, limit int, zone, sort, filterBy, typ string, hasMagnets bool) (pipeline.Envelope, error) {
	keyword := pipeline.ConsumerRef(input)
	opt := javdb.SearchOptions{
		Page:     page,
		Limit:    limit,
		Zone:     zone,
		Sort:     sort,
		FilterBy: filterBy,
		Type:     typ,
	}
	res, err := c.Search(ctx, keyword, opt)
	if err != nil {
		return pipeline.Envelope{}, fmt.Errorf("search failed: %w", err)
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = result.FilterMoviesWithMagnets(movies)
		}
		movieID := ""
		if len(movies) > 0 {
			movieID = fmt.Sprint(movies[0]["id"])
		}
		envelope := pipeline.New(pipeline.KindMovie, keyword, movieID)
		if len(movies) > 0 {
			envelope = envelope.WithData(map[string]any{"movies": movies})
		}
		return envelope, nil
	}
	key := searchTypeKey(typ)
	items := res.Named(key)
	envelope := pipeline.New(pipeline.Kind(typ), keyword, "")
	if len(items) > 0 {
		envelope = envelope.WithData(map[string]any{key: items})
	}
	return envelope, nil
}

// runSearchMany 执行单个关键词搜索并返回多个信封（fan-out）：
//   - movie 类型：每部影片一个 KindMovie 信封，ref 为番号，id 为内部 ID。
//   - 其他 --type：回退为单信封（与 runSearchOne 相同）。
func runSearchMany(c *javdb.Client, ctx context.Context, input pipeline.Envelope, page, limit int, zone, sort, filterBy, typ string, hasMagnets bool) ([]pipeline.Envelope, error) {
	keyword := pipeline.ConsumerRef(input)
	opt := javdb.SearchOptions{
		Page:     page,
		Limit:    limit,
		Zone:     zone,
		Sort:     sort,
		FilterBy: filterBy,
		Type:     typ,
	}
	res, err := c.Search(ctx, keyword, opt)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = result.FilterMoviesWithMagnets(movies)
		}
		envelopes := make([]pipeline.Envelope, 0, len(movies))
		for _, m := range movies {
			ref := fmt.Sprint(m["number"])
			id := fmt.Sprint(m["id"])
			envelope := pipeline.New(pipeline.KindMovie, ref, id).WithData(map[string]any{"movie": m})
			envelopes = append(envelopes, envelope)
		}
		return envelopes, nil
	}
	// 非 movie 类型回退为单信封。
	one, err := runSearchOne(c, ctx, input, page, limit, zone, sort, filterBy, typ, hasMagnets)
	if err != nil {
		return nil, err
	}
	return []pipeline.Envelope{one}, nil
}
func runTextSearch(options *invocation.RootOptions, streams *invocation.Streams, keyword string, page, limit int, zone, sort, filterBy, typ string, hasMagnets, asJSON bool) error {
	c, err := client.New(options, "")
	if err != nil {
		return err
	}
	opt := javdb.SearchOptions{
		Page:     page,
		Limit:    limit,
		Zone:     zone,
		Sort:     sort,
		FilterBy: filterBy,
		Type:     typ,
	}
	res, err := c.Search(context.Background(), keyword, opt)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = result.FilterMoviesWithMagnets(movies)
		}
		if asJSON {
			return writeJSON(streams.Out, map[string]any{"movies": movies})
		}
		return writeMovieRows(streams.Out, streams.Err, movies)
	}
	key := searchTypeKey(typ)
	items := res.Named(key)
	if asJSON {
		return writeJSON(streams.Out, map[string]any{key: items})
	}
	return writeNamedRows(streams.Out, streams.Err, items)
}

// runImageSearch 执行以图搜番：读取并校验原始图片（路径/URL/stdin），调用
// 公开 SDK 反搜+严格联动，输出候选、相似度、帧、匹配详情与逐项错误。
// provider 顶层失败直接报错；候选部分失败在完成输出后返回非零。
func runImageSearch(options *invocation.RootOptions, streams *invocation.Streams, arg string, reader *bufio.Reader, source string, noCache bool, mode pipeline.OutputMode) error {
	setup, err := client.NewReverseSearchClient(options, "", source)
	if err != nil {
		return err
	}
	imageBytes, err := readInputImage(arg, reader, setup.HTTPClient)
	if err != nil {
		return err
	}
	result, err := setup.Client.SearchByImage(context.Background(), javdb.ReverseSearchRequest{
		Image:       imageBytes.Bytes,
		Filename:    imageBytes.Filename,
		Source:      setup.Source,
		BypassCache: noCache,
	}, javdb.ImageSearchOptions{})
	if err != nil {
		// provider 顶层失败：不伪造空结果。
		return fmt.Errorf("reverse search failed: %w", err)
	}
	switch mode {
	case pipeline.OutputJSON:
		if err := writeJSON(streams.Out, map[string]any{
			"reverse_search": result.ReverseSearch,
			"matches":        result.Matches,
		}); err != nil {
			return err
		}
	case pipeline.OutputNDJSON:
		writer := pipeline.NewWriter(streams.Out, pipeline.OutputNDJSON)
		for _, match := range result.Matches {
			if match.Error != nil {
				if err := writer.Write(pipeline.ErrorEnvelope(pipeline.New(pipeline.KindMovie, match.Candidate.VideoCode, match.MovieID), "search", "link", match.Error.Code, match.Error.Message)); err != nil {
					return err
				}
				continue
			}
			envelope := pipeline.New(pipeline.KindMovie, match.Candidate.VideoCode, match.MovieID).
				WithData(map[string]any{"movie": match.Movie, "similarity": match.Candidate.Similarity}).
				WithMeta(map[string]any{"reverse_search": map[string]any{"source": result.ReverseSearch.Source, "candidates": result.ReverseSearch.Candidates}})
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
	default:
		if err := writeImageSearchText(streams.Out, result); err != nil {
			return err
		}
	}
	if failures := countImageFailures(result); failures > 0 {
		return fmt.Errorf("reverse search completed with %d of %d candidates failed", failures, len(result.Matches))
	}
	return nil
}

func readInputImage(arg string, reader *bufio.Reader, httpClient *http.Client) (*image.Image, error) {
	if arg == "" {
		return image.ReadStream(reader, "<stdin>")
	}
	if isHTTPURL(arg) {
		// 复用最终代理配置（与 provider/JavDB 请求一致）；nil 时 image 包回退
		// http.DefaultClient。
		return image.ReadURL(context.Background(), httpClient, arg)
	}
	return image.ReadFile(arg)
}

func countImageFailures(result javdb.ImageSearchResult) int {
	failures := 0
	for _, match := range result.Matches {
		if match.Error != nil {
			failures++
		}
	}
	return failures
}

// writeImageSearchText 人类可读输出：候选行、相似度、帧、严格匹配详情与逐项错误。
func writeImageSearchText(w io.Writer, searchResult javdb.ImageSearchResult) error {
	for index, match := range searchResult.Matches {
		candidate := match.Candidate
		if match.Error != nil {
			if _, err := fmt.Fprintf(w, "%d. %s (%.1f%%)  [失败: %s]\n", index+1, candidate.VideoCode, candidate.Similarity, match.Error.Message); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%d. %s (%.1f%%)\n", index+1, candidate.VideoCode, candidate.Similarity); err != nil {
			return err
		}
		for _, frame := range candidate.Frames {
			line := fmt.Sprintf("   帧 %s 相似度 %.1f%%", frame.Timestamp, frame.Similarity)
			if frame.ThumbnailURL != "" {
				line += "  " + frame.ThumbnailURL
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
		if match.Movie != nil {
			rows := result.ProjectMovies([]map[string]any{match.Movie})
			for _, row := range rows {
				if _, err := fmt.Fprintf(w, "   => %s\n", row.Line()); err != nil {
					return err
				}
			}
		}
	}
	if len(searchResult.Matches) == 0 {
		_, err := fmt.Fprintln(w, "(无候选)")
		return err
	}
	return nil
}

// searchTypeKey 将 --type 映射为搜索响应 list key（movie/空 → movies）。
func searchTypeKey(type_ string) string {
	switch type_ {
	case "movie", "":
		return "movies"
	case "code":
		return "codes"
	case "series":
		return "series"
	case "actor":
		return "actors"
	case "maker":
		return "makers"
	case "director":
		return "directors"
	case "list":
		return "lists"
	default:
		return "movies"
	}
}

// writeMovieRows 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovieRows(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectMovies(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}

// writeNamedRows 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamedRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectNamedAll(items) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// runSearchMagnets 执行关键词搜索并对每部影片获取磁力：筛选 → 排序 → 截取 N。
// 结果按搜索结果顺序（影片）和磁力排序顺序输出。
func runSearchMagnets(options *invocation.RootOptions, streams *invocation.Streams, inputs []pipeline.Envelope, mode pipeline.OutputMode, page, limit int, zone, sort, filterBy string, hasMagnets bool, magnetsCount int, cnsub, hd bool, minSize string) error {
	c, err := client.New(options, "")
	if err != nil {
		return err
	}
	minMiB := 0
	if minSize != "" {
		minMiB, err = magnetspkg.ParseSizeMiB(minSize)
		if err != nil {
			return err
		}
	}

	// 收集所有搜索结果影片（保留输入顺序）。
	var allMovies []map[string]any
	for _, input := range inputs {
		keyword := pipeline.ConsumerRef(input)
		opt := javdb.SearchOptions{
			Page: page, Limit: limit, Zone: zone, Sort: sort, FilterBy: filterBy, Type: "movie",
		}
		res, err := c.Search(context.Background(), keyword, opt)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		movies := res.Movies()
		if hasMagnets {
			movies = result.FilterMoviesWithMagnets(movies)
		}
		allMovies = append(allMovies, movies...)
	}

	// 并发获取每部影片的磁力。
	type magnetResult struct {
		movie   map[string]any
		magnets []map[string]any
		err     error
	}
	results := make([]magnetResult, len(allMovies))
	var wait sync.WaitGroup
	for i := range allMovies {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			mid := fmt.Sprint(allMovies[i]["id"])
			mags, err := c.MovieMagnets(context.Background(), mid)
			if err != nil {
				results[i] = magnetResult{movie: allMovies[i], err: err}
				return
			}
			mags = javdb.FilterMagnets(mags, cnsub, hd, minMiB)
			mags = javdb.RankMagnets(mags, magnetsCount)
			results[i] = magnetResult{movie: allMovies[i], magnets: mags}
		}(i)
	}
	wait.Wait()

	// 输出。
	failures := 0
	switch mode {
	case pipeline.OutputJSON:
		moviesOut := make([]map[string]any, 0, len(results))
		for _, r := range results {
			entry := map[string]any{
				"number":  r.movie["number"],
				"id":      r.movie["id"],
				"magnets": r.magnets,
			}
			if r.err != nil {
				entry["error"] = r.err.Error()
			}
			moviesOut = append(moviesOut, entry)
		}
		return writeJSON(streams.Out, map[string]any{"movies": moviesOut})
	case pipeline.OutputNDJSON:
		writer := pipeline.NewWriter(streams.Out, pipeline.OutputNDJSON)
		for _, r := range results {
			ref := fmt.Sprint(r.movie["number"])
			id := fmt.Sprint(r.movie["id"])
			if r.err != nil {
				if err := writer.Write(pipeline.ErrorEnvelope(pipeline.New(pipeline.KindMagnet, ref, id), "search", "magnets", "fetch", r.err.Error())); err != nil {
					return err
				}
				failures++
				continue
			}
			envelope := pipeline.New(pipeline.KindMagnet, ref, id).
				WithData(map[string]any{"movie": r.movie, "magnets": r.magnets})
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
	default:
		// 文本/人类模式：逐磁力 URI 输出，失败项写 stderr。
		for _, r := range results {
			if r.err != nil {
				failures++
				fmt.Fprintf(streams.Err, "search: %s\n", r.err.Error())
				continue
			}
			for _, m := range r.magnets {
				uri := javdb.MagnetURI(m)
				if uri != "" {
					fmt.Fprintln(streams.Out, uri)
				}
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("search --magnets completed with %d of %d movies failed", failures, len(results))
	}
	return nil
}

// runImageSearchMagnets 执行以图搜番并直接获取磁力：使用 SkipMovieDetail 跳过
// 完整详情，仅做番号→ID→磁力。结果按 provider 候选顺序输出。
func runImageSearchMagnets(options *invocation.RootOptions, streams *invocation.Streams, arg string, reader *bufio.Reader, source string, noCache bool, mode pipeline.OutputMode, magnetsCount int, cnsub, hd bool, minSize string) error {
	setup, err := client.NewReverseSearchClient(options, "", source)
	if err != nil {
		return err
	}
	imageBytes, err := readInputImage(arg, reader, setup.HTTPClient)
	if err != nil {
		return err
	}
	result, err := setup.Client.SearchByImage(context.Background(), javdb.ReverseSearchRequest{
		Image:       imageBytes.Bytes,
		Filename:    imageBytes.Filename,
		Source:      setup.Source,
		BypassCache: noCache,
	}, javdb.ImageSearchOptions{SkipMovieDetail: true})
	if err != nil {
		return fmt.Errorf("reverse search failed: %w", err)
	}

	minMiB := 0
	if minSize != "" {
		minMiB, err = magnetspkg.ParseSizeMiB(minSize)
		if err != nil {
			return err
		}
	}

	// 并发获取每个候选的磁力。
	type magnetResult struct {
		match   javdb.ImageSearchMatch
		magnets []map[string]any
		err     error
	}
	results := make([]magnetResult, len(result.Matches))
	var wait sync.WaitGroup
	for i := range result.Matches {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			match := result.Matches[i]
			if match.Error != nil {
				results[i] = magnetResult{match: match, err: fmt.Errorf("%s", match.Error.Message)}
				return
			}
			if match.MovieID == "" {
				results[i] = magnetResult{match: match, err: fmt.Errorf("no movie id")}
				return
			}
			mags, err := setup.Client.MovieMagnets(context.Background(), match.MovieID)
			if err != nil {
				results[i] = magnetResult{match: match, err: err}
				return
			}
			mags = javdb.FilterMagnets(mags, cnsub, hd, minMiB)
			mags = javdb.RankMagnets(mags, magnetsCount)
			results[i] = magnetResult{match: match, magnets: mags}
		}(i)
	}
	wait.Wait()

	failures := 0
	switch mode {
	case pipeline.OutputJSON:
		matchesOut := make([]map[string]any, 0, len(results))
		for _, r := range results {
			entry := map[string]any{
				"video_code": r.match.Candidate.VideoCode,
				"movie_id":   r.match.MovieID,
				"magnets":    r.magnets,
			}
			if r.err != nil {
				entry["error"] = r.err.Error()
			}
			matchesOut = append(matchesOut, entry)
		}
		return writeJSON(streams.Out, map[string]any{"reverse_search": result.ReverseSearch, "matches": matchesOut})
	case pipeline.OutputNDJSON:
		writer := pipeline.NewWriter(streams.Out, pipeline.OutputNDJSON)
		for _, r := range results {
			ref := r.match.Candidate.VideoCode
			id := r.match.MovieID
			if r.err != nil {
				if err := writer.Write(pipeline.ErrorEnvelope(pipeline.New(pipeline.KindMagnet, ref, id), "search", "magnets", "fetch", r.err.Error())); err != nil {
					return err
				}
				failures++
				continue
			}
			envelope := pipeline.New(pipeline.KindMagnet, ref, id).
				WithData(map[string]any{
					"movie":      map[string]any{"id": r.match.MovieID},
					"magnets":    r.magnets,
					"similarity": r.match.Candidate.Similarity,
				}).
				WithMeta(map[string]any{"reverse_search": map[string]any{"source": result.ReverseSearch.Source, "candidates": result.ReverseSearch.Candidates}})
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
	default:
		for _, r := range results {
			if r.err != nil {
				failures++
				fmt.Fprintf(streams.Err, "search: %s\n", r.err.Error())
				continue
			}
			for _, m := range r.magnets {
				uri := javdb.MagnetURI(m)
				if uri != "" {
					fmt.Fprintln(streams.Out, uri)
				}
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("search --magnets completed with %d of %d candidates failed", failures, len(results))
	}
	return nil
}
