// Package search 提供影片/实体搜索命令与以图搜番图片模式。
package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/spf13/cobra"

	magnetspkg "github.com/FlanChanXwO/javdb-cli/internal/cli/magnets"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/logging"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/image"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the movie/dimension search command and the image reverse search mode.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
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
		Short: i18n.T("command.search"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := record.ResolveOutputMode(asNDJSON, asJSON, streams.OutIsTerminal)
			if err != nil {
				return err
			}
			if err := validateZone(zone); err != nil {
				return err
			}
			if err := validateSearchType(typ); err != nil {
				return err
			}
			if magnets < 0 {
				return i18n.Errorf("search.invalid_magnet_count")
			}
			filtersChanged := cmd.Flags().Changed("cnsub") || cmd.Flags().Changed("hd") || cmd.Flags().Changed("min-size")
			if filtersChanged && !cmd.Flags().Changed("magnets") {
				return i18n.Errorf("search.magnet_filters_require_magnets")
			}
			// --magnets 仅支持 movie 搜索。
			if magnets > 0 || cmd.Flags().Changed("magnets") {
				if typ != "" && typ != "movie" {
					return fmt.Errorf(i18n.T("search.magnets_require_movie"), typ)
				}
			}
			minMiB := 0
			if cmd.Flags().Changed("magnets") && minSize != "" {
				minMiB, err = magnetspkg.ParseSizeMiB(minSize)
				if err != nil {
					// 这是纯本地 flag 校验，必须先于图片读取、上传和 JavDB 联动。
					return err
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
					return runImageSearchMagnets(cmd.Context(), options, streams, arg, reader, source, noCache, mode, magnets, cnsub, hd, minMiB)
				}
				return runImageSearch(cmd.Context(), options, streams, arg, reader, source, noCache, mode)
			}
			if len(inputs) == 0 {
				return i18n.Errorf("input.search_required")
			}
			if cmd.Flags().Changed("magnets") {
				return runSearchMagnets(cmd.Context(), options, streams, inputs, mode, page, limit, zone, sort, filterBy, hasMagnets, magnets, cnsub, hd, minMiB)
			}
			runner := &record.BatchRunner{
				Name:       "search",
				Context:    cmd.Context(),
				LegacyJSON: true,
				// 非 TTY 单项也输出可供下游消费的稳定番号记录。
				RouteTextThroughPipeline: true,
				Kinds:                    []record.Kind{record.KindMovie},
				ClientFactory: func() (*session.Client, error) {
					return session.NewWithDefaultToken(options)
				},
				RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
					return runSearchOne(c, ctx, input, page, limit, zone, sort, filterBy, typ, hasMagnets)
				},
				RunMany: func(c *session.Client, ctx context.Context, input record.Envelope) ([]record.Envelope, error) {
					return runSearchMany(c, ctx, input, page, limit, zone, sort, filterBy, typ, hasMagnets)
				},
				Legacy: func(args []string) error {
					return runTextSearch(cmd.Context(), options, streams, args[0], page, limit, zone, sort, filterBy, typ, hasMagnets, asJSON)
				},
			}
			return runner.ExecuteWithInputs(streams, inputs, mode)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 0, i18n.T("flag.search_limit"))
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2|all")
	cmd.Flags().StringVar(&sort, "sort", "", "relevance|release|score|update|hit")
	cmd.Flags().StringVar(&filterBy, "filter-by", "", "can_play|magnets|subtitle|single")
	cmd.Flags().StringVar(&typ, "type", "", "movie|code|series|actor|maker|director|list")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("flag.has_magnets"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	cmd.Flags().BoolVar(&asImage, "image", false, i18n.T("flag.image"))
	cmd.Flags().StringVar(&source, "source", "", i18n.T("flag.image_source"))
	cmd.Flags().BoolVar(&noCache, "no-cache", false, i18n.T("flag.no_cache"))
	cmd.Flags().IntVar(&magnets, "magnets", 0, i18n.T("flag.search_magnets"))
	cmd.Flags().BoolVar(&cnsub, "cnsub", false, i18n.T("flag.search_cnsub"))
	cmd.Flags().BoolVar(&hd, "hd", false, i18n.T("flag.search_hd"))
	cmd.Flags().StringVar(&minSize, "min-size", "", i18n.T("flag.search_min_size"))
	return cmd
}

func validateZone(zone string) error {
	switch zone {
	case "censored", "uncensored", "western", "fc2", "all":
		return nil
	default:
		return fmt.Errorf(i18n.T("search.invalid_zone"), zone)
	}
}

func validateSearchType(value string) error {
	switch value {
	case "", "movie", "code", "series", "actor", "maker", "director", "list":
		return nil
	default:
		return fmt.Errorf(i18n.T("search.invalid_type"), value)
	}
}

// classifySearchInput 一次性分类输入：返回图片模式标志与批处理输入。
//   - --image 强制图片；参数是现有普通文件或 HTTP(S) URL 自动图片。
//   - 无参数非 TTY stdin 按固定顺序分类（图片 magic → NDJSON → 文本）。
//   - 位置参数与非空 stdin 同时存在是歧义错误，绝不静默丢数据。
func classifySearchInput(arg string, forced bool, reader *bufio.Reader, streams *environment.Streams) (bool, []record.Envelope, error) {
	stdinHasContent := false
	if !streams.InIsTerminal {
		if _, err := reader.Peek(1); err == nil {
			stdinHasContent = true
		} else if !errors.Is(err, io.EOF) {
			return false, nil, fmt.Errorf(i18n.T("input.read_failed"), err)
		}
	}
	if arg != "" && stdinHasContent {
		return false, nil, i18n.Errorf("input.ambiguous")
	}
	if arg != "" {
		if forced || isHTTPURL(arg) || isRegularFile(arg) {
			return true, nil, nil
		}
		return false, []record.Envelope{record.New("", arg, "")}, nil
	}
	if streams.InIsTerminal || !stdinHasContent {
		return false, nil, nil
	}
	classification, content, err := record.Classify(reader)
	if err != nil {
		return false, nil, err
	}
	switch classification {
	case record.ClassificationImage:
		return true, nil, nil
	case record.ClassificationNDJSON:
		inputs, err := record.ParseBatch(content, record.KindNDJSONInput)
		return false, inputs, err
	default:
		inputs, err := record.ParseBatch(content, record.KindMovie)
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
func runSearchOne(c *session.Client, ctx context.Context, input record.Envelope, page, limit int, zone, sort, filterBy, typ string, hasMagnets bool) (record.Envelope, error) {
	keyword := record.ConsumerRef(input)
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
		return record.Envelope{}, err
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = display.FilterMoviesWithMagnets(movies)
		}
		movieID := ""
		if len(movies) > 0 {
			movieID = fmt.Sprint(movies[0]["id"])
		}
		envelope := record.New(record.KindMovie, keyword, movieID)
		if len(movies) > 0 {
			envelope = envelope.WithData(map[string]any{"movies": movies})
		}
		return envelope, nil
	}
	key := searchTypeKey(typ)
	items := res.Named(key)
	envelope := record.New(record.Kind(typ), keyword, "")
	if len(items) > 0 {
		envelope = envelope.WithData(map[string]any{key: items})
	}
	return envelope, nil
}

// runSearchMany 执行单个关键词搜索并返回多个信封（fan-out）：
//   - movie 类型：每部影片一个 KindMovie 信封，ref 为番号，id 为内部 ID。
//   - 其他 --type：每个命名实体一个对应 kind 的信封。
func runSearchMany(c *session.Client, ctx context.Context, input record.Envelope, page, limit int, zone, sort, filterBy, typ string, hasMagnets bool) ([]record.Envelope, error) {
	keyword := record.ConsumerRef(input)
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
		return nil, err
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = display.FilterMoviesWithMagnets(movies)
		}
		envelopes := make([]record.Envelope, 0, len(movies))
		for _, m := range movies {
			ref := fmt.Sprint(m["number"])
			id := fmt.Sprint(m["id"])
			envelope := record.New(record.KindMovie, ref, id).WithData(map[string]any{"movie": m})
			envelopes = append(envelopes, envelope)
		}
		return envelopes, nil
	}
	key := searchTypeKey(typ)
	items := res.Named(key)
	envelopes := make([]record.Envelope, 0, len(items))
	for _, item := range items {
		ref, id := namedEntityRef(item)
		envelope := record.New(record.Kind(typ), ref, id).WithData(map[string]any{typ: item})
		envelopes = append(envelopes, envelope)
	}
	return envelopes, nil
}

// namedEntityRef 为命名搜索结果选择可读且稳定的引用，同时保留内部 ID。
func namedEntityRef(item map[string]any) (string, string) {
	id := scalar.String(item["id"])
	for _, key := range []string{"name_zht", "name", "code", "number"} {
		if ref := scalar.String(item[key]); ref != "" {
			return ref, id
		}
	}
	return id, id
}
func runTextSearch(ctx context.Context, options *session.Options, streams *environment.Streams, keyword string, page, limit int, zone, sort, filterBy, typ string, hasMagnets, asJSON bool) error {
	c, err := session.NewWithDefaultToken(options)
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
	res, err := c.Search(ctx, keyword, opt)
	if err != nil {
		return err
	}
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = display.FilterMoviesWithMagnets(movies)
		}
		if asJSON {
			return writeJSON(streams.Out, map[string]any{"movies": movies})
		}
		return writeMovieRows(streams.Out, streams.Diagnostics(slog.LevelInfo), movies)
	}
	key := searchTypeKey(typ)
	items := res.Named(key)
	if asJSON {
		return writeJSON(streams.Out, map[string]any{key: items})
	}
	return writeNamedRows(streams.Out, streams.Diagnostics(slog.LevelInfo), items)
}

// runImageSearch 执行以图搜番：读取并校验原始图片（路径/URL/stdin），调用
// 公开 SDK 反搜+严格联动，输出候选、相似度、帧、匹配详情与逐项错误。
// provider 顶层失败直接报错；候选部分失败在完成输出后返回非零。
func runImageSearch(ctx context.Context, options *session.Options, streams *environment.Streams, arg string, reader *bufio.Reader, source string, noCache bool, mode record.OutputMode) error {
	setup, err := session.NewReverseSearchClient(options, source)
	if err != nil {
		return err
	}
	imageBytes, err := readInputImage(ctx, arg, reader, setup.HTTPClient)
	if err != nil {
		return err
	}
	result, err := setup.Client.SearchByImage(ctx, javdb.ReverseSearchRequest{
		Image:       imageBytes.Bytes,
		Filename:    imageBytes.Filename,
		Source:      setup.Source,
		BypassCache: noCache,
	}, javdb.ImageSearchOptions{SkipMovieDetail: mode == record.OutputText})
	if err != nil {
		// provider 顶层失败：不伪造空结果。
		return fmt.Errorf(i18n.T("image.search_failed"), err)
	}
	switch mode {
	case record.OutputJSON:
		if err := writeJSON(streams.Out, map[string]any{
			"reverse_search": result.ReverseSearch,
			"matches":        result.Matches,
		}); err != nil {
			return err
		}
	case record.OutputNDJSON:
		writer := record.NewWriter(streams.Out, record.OutputNDJSON)
		for _, match := range result.Matches {
			if match.Error != nil {
				if err := writer.Write(record.ErrorEnvelope(record.New(record.KindMovie, match.Candidate.VideoCode, match.MovieID), "search", "link", match.Error.Code, match.Error.Message)); err != nil {
					return err
				}
				continue
			}
			envelope := record.New(record.KindMovie, match.Candidate.VideoCode, match.MovieID).
				WithData(map[string]any{"movie": match.Movie, "similarity": match.Candidate.Similarity}).
				WithMeta(map[string]any{"reverse_search": map[string]any{"source": result.ReverseSearch.Source, "candidates": result.ReverseSearch.Candidates}})
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
	case record.OutputHuman:
		if err := writeImageSearchText(streams.Out, streams.Diagnostics(slog.LevelError), result); err != nil {
			return err
		}
	case record.OutputText:
		// 非 TTY stdout 只输出成功影片番号；候选、帧和失败诊断全部写 stderr，
		// 避免下游 magnets 将人类描述误当作输入番号。
		if err := writeImageSearchRecords(streams.Out, result); err != nil {
			return err
		}
		if err := writeImageSearchText(streams.Diagnostics(slog.LevelInfo), streams.Diagnostics(slog.LevelError), result); err != nil {
			return err
		}
	}
	if failures := countImageFailures(result); failures > 0 {
		return fmt.Errorf(i18n.T("image.partial_failure"), failures, len(result.Matches))
	}
	return nil
}

func readInputImage(ctx context.Context, arg string, reader *bufio.Reader, httpClient *http.Client) (*image.Image, error) {
	if arg == "" {
		return image.ReadStream(reader, "<stdin>")
	}
	if isHTTPURL(arg) {
		// 复用最终代理配置（与 provider/JavDB 请求一致）；nil 时 image 包回退
		// http.DefaultClient。
		return image.ReadURL(ctx, httpClient, arg)
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

// writeImageSearchRecords 输出非 TTY 可消费的成功影片番号，一行一个记录。
func writeImageSearchRecords(w io.Writer, searchResult javdb.ImageSearchResult) error {
	for index, match := range searchResult.Matches {
		if match.Error != nil {
			continue
		}
		ref := match.Candidate.VideoCode
		if ref == "" {
			return fmt.Errorf(i18n.T("image.missing_video_code"), index+1)
		}
		if _, err := fmt.Fprintln(w, ref); err != nil {
			return err
		}
	}
	return nil
}

// writeImageSearchText 输出人类诊断：候选行、相似度、帧、严格匹配详情与逐项错误。
// infoW 用于成功候选的诊断，errW 专用于失败项；调用方可将二者都指向 stderr
// 以保持非 TTY stdout 的机器记录契约。
func writeImageSearchText(infoW, errW io.Writer, searchResult javdb.ImageSearchResult) error {
	for index, match := range searchResult.Matches {
		candidate := match.Candidate
		if match.Error != nil {
			if _, err := fmt.Fprintf(errW, i18n.T("image.candidate_failed"), index+1, candidate.VideoCode, candidate.Similarity, match.Error.Message); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(infoW, "%d. %s (%.1f%%)\n", index+1, candidate.VideoCode, candidate.Similarity); err != nil {
			return err
		}
		for _, frame := range candidate.Frames {
			line := fmt.Sprintf(i18n.T("image.frame"), frame.Timestamp, frame.Similarity)
			if frame.ThumbnailURL != "" {
				line += "  " + frame.ThumbnailURL
			}
			if _, err := fmt.Fprintln(infoW, line); err != nil {
				return err
			}
		}
		if match.Movie != nil {
			rows := display.ProjectMovies([]map[string]any{match.Movie})
			for _, row := range rows {
				if _, err := fmt.Fprintf(infoW, "   => %s\n", row.Line()); err != nil {
					return err
				}
			}
		}
	}
	if len(searchResult.Matches) == 0 {
		_, err := fmt.Fprintln(infoW, i18n.T("image.no_candidates"))
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
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for index, row := range display.ProjectMovies(movies) {
		line := row.Line()
		if minutes, ok := display.MovieDurationMinutes(movies[index]); ok {
			line += fmt.Sprintf("\t%dm", minutes)
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// writeNamedRows 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamedRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for _, row := range display.ProjectNamedAll(items) {
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

// movieFromEnvelope 将已解析的 movie 信封投影为磁力请求所需的最小 movie
// 对象；保留上游详情字段，同时确保 number/id 缺失时仍可用信封字段补齐。
func movieFromEnvelope(input record.Envelope) map[string]any {
	movie := make(map[string]any)
	if raw, ok := input.Data["movie"].(map[string]any); ok {
		for key, value := range raw {
			movie[key] = value
		}
	}
	if scalar.String(movie["number"]) == "" {
		movie["number"] = input.Ref
	}
	if scalar.String(movie["id"]) == "" {
		movie["id"] = input.ID
	}
	return movie
}

// runSearchMagnets 执行关键词搜索并对每部影片获取磁力：筛选 → 排序 → 截取 N。
// 结果按搜索结果顺序（影片）和磁力排序顺序输出。
func runSearchMagnets(ctx context.Context, options *session.Options, streams *environment.Streams, inputs []record.Envelope, mode record.OutputMode, page, limit int, zone, sort, filterBy string, hasMagnets bool, magnetsCount int, cnsub, hd bool, minMiB int) error {
	c, err := session.NewWithDefaultToken(options)
	if err != nil {
		return err
	}

	type magnetResult struct {
		input         record.Envelope
		movie         map[string]any
		magnets       []map[string]any
		err           error
		errorEnvelope record.Envelope
	}

	// 输入校验与搜索结果使用同一保序序列。错误信封不参与搜索，避免放大
	// 上游失败；非法 kind 也在发出远程请求前转为原位错误。
	var results []magnetResult
	for _, input := range inputs {
		if input.Kind == record.KindError {
			results = append(results, magnetResult{
				input:         input,
				err:           errors.New(record.ErrorMessage(input)),
				errorEnvelope: input,
			})
			continue
		}
		if input.Kind != "" && input.Kind != record.KindMovie {
			itemErr := fmt.Errorf(i18n.T("record.unsupported_kind"), input.Kind)
			results = append(results, magnetResult{
				input:         input,
				err:           itemErr,
				errorEnvelope: record.ErrorEnvelope(input, "search", "input", "kind", itemErr.Error()),
			})
			continue
		}
		if input.Kind == record.KindMovie && input.ID != "" {
			// search --ndjson | search --magnets --json：上游已经解析出影片，
			// 直接沿用其 ID，不能把内部 ID 当作新的番号关键词再搜索一次。
			results = append(results, magnetResult{
				input: input,
				movie: movieFromEnvelope(input),
			})
			continue
		}
		keyword := record.ConsumerRef(input)
		opt := javdb.SearchOptions{
			Page: page, Limit: limit, Zone: zone, Sort: sort, FilterBy: filterBy, Type: "movie",
		}
		res, err := c.Search(ctx, keyword, opt)
		if err != nil {
			itemErr := err
			results = append(results, magnetResult{
				input:         input,
				err:           itemErr,
				errorEnvelope: record.ErrorEnvelope(input, "search", "search", "fetch", itemErr.Error()),
			})
			continue
		}
		movies := res.Movies()
		if hasMagnets {
			movies = display.FilterMoviesWithMagnets(movies)
		}
		for _, movie := range movies {
			results = append(results, magnetResult{input: input, movie: movie})
		}
	}

	// 使用有界 worker 池获取磁力，避免结果量放大为无界 API 并发；索引写入
	// 保留搜索结果顺序，ctx 取消时不再领取新任务。
	const workers = 4
	jobs := make(chan int)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := range jobs {
				if results[i].err != nil {
					continue
				}
				mid := fmt.Sprint(results[i].movie["id"])
				mags, err := c.MovieMagnets(ctx, mid)
				if err != nil {
					results[i].err = err
					results[i].errorEnvelope = record.ErrorEnvelope(
						record.New(record.KindMagnet, fmt.Sprint(results[i].movie["number"]), mid),
						"search", "magnets", "fetch", err.Error(),
					)
					continue
				}
				mags = javdb.FilterMagnets(mags, cnsub, hd, minMiB)
				results[i].magnets = javdb.RankMagnets(mags, magnetsCount)
			}
		}()
	}
	for i := range results {
		if results[i].err == nil {
			select {
			case jobs <- i:
			case <-ctx.Done():
				close(jobs)
				wait.Wait()
				return ctx.Err()
			}
		}
	}
	close(jobs)
	wait.Wait()

	// 输出。
	failures := 0
	switch mode {
	case record.OutputJSON:
		if len(inputs) > 1 {
			// 批量 JSON 遵循 pipeline cardinality：每个影片/错误都是一个
			// envelope，并在 meta 中保留产生它的输入关键词，不能再扁平化成
			// 一个没有来源信息的 {"movies": [...]} 对象。
			batchOut := make([]record.Envelope, 0, len(results))
			for _, r := range results {
				if r.err != nil {
					failures++
				}
				if r.movie == nil || r.err != nil {
					batchOut = append(batchOut, withInputMeta(r.errorEnvelope, r.input))
					continue
				}
				ref := fmt.Sprint(r.movie["number"])
				id := fmt.Sprint(r.movie["id"])
				batchOut = append(batchOut, withInputMeta(
					record.New(record.KindMagnet, ref, id).
						WithData(map[string]any{"movie": r.movie, "magnets": r.magnets}),
					r.input,
				))
			}
			if err := writeJSON(streams.Out, batchOut); err != nil {
				return err
			}
			break
		}
		moviesOut := make([]map[string]any, 0, len(results))
		for _, r := range results {
			if r.movie == nil {
				failures++
				moviesOut = append(moviesOut, map[string]any{"error": r.err.Error(), "envelope": r.errorEnvelope})
				continue
			}
			entry := map[string]any{
				"number":  r.movie["number"],
				"id":      r.movie["id"],
				"magnets": r.magnets,
			}
			if r.err != nil {
				failures++
				entry["error"] = r.err.Error()
			}
			moviesOut = append(moviesOut, entry)
		}
		if err := writeJSON(streams.Out, map[string]any{"movies": moviesOut}); err != nil {
			return err
		}
	case record.OutputNDJSON:
		writer := record.NewWriter(streams.Out, record.OutputNDJSON)
		for _, r := range results {
			if r.movie == nil {
				if err := writer.Write(r.errorEnvelope); err != nil {
					return err
				}
				failures++
				continue
			}
			ref := fmt.Sprint(r.movie["number"])
			id := fmt.Sprint(r.movie["id"])
			if r.err != nil {
				if err := writer.Write(r.errorEnvelope); err != nil {
					return err
				}
				failures++
				continue
			}
			envelope := record.New(record.KindMagnet, ref, id).
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
				continue
			}
			for _, m := range r.magnets {
				uri := javdb.MagnetURI(m)
				if uri != "" {
					if _, err := fmt.Fprintln(streams.Out, uri); err != nil {
						return err
					}
				}
			}
		}
	}
	if failures > 0 {
		var causes []error
		for _, r := range results {
			if r.err != nil {
				causes = append(causes, r.err)
				if err := streams.Failure(r.err); err != nil {
					return err
				}
			}
		}
		return &logging.Reported{Failed: failures, Total: len(results), Cause: errors.Join(causes...)}
	}
	return nil
}

func withInputMeta(envelope, input record.Envelope) record.Envelope {
	meta := make(map[string]any, len(envelope.Meta)+2)
	for key, value := range envelope.Meta {
		meta[key] = value
	}
	if input.Ref != "" {
		meta["input_ref"] = input.Ref
	}
	if input.ID != "" {
		meta["input_id"] = input.ID
	}
	if len(meta) == 0 {
		return envelope
	}
	return envelope.WithMeta(meta)
}

// runImageSearchMagnets 执行以图搜番并直接获取磁力：使用 SkipMovieDetail 跳过
// 完整详情，仅做番号→ID→磁力。结果按 provider 候选顺序输出。
func runImageSearchMagnets(ctx context.Context, options *session.Options, streams *environment.Streams, arg string, reader *bufio.Reader, source string, noCache bool, mode record.OutputMode, magnetsCount int, cnsub, hd bool, minMiB int) error {
	setup, err := session.NewReverseSearchClient(options, source)
	if err != nil {
		return err
	}
	imageBytes, err := readInputImage(ctx, arg, reader, setup.HTTPClient)
	if err != nil {
		return err
	}
	result, err := setup.Client.SearchByImage(ctx, javdb.ReverseSearchRequest{
		Image:       imageBytes.Bytes,
		Filename:    imageBytes.Filename,
		Source:      setup.Source,
		BypassCache: noCache,
	}, javdb.ImageSearchOptions{SkipMovieDetail: true})
	if err != nil {
		return fmt.Errorf(i18n.T("image.search_failed"), err)
	}

	// 使用有界 worker 池获取每个候选的磁力，并按候选索引保序。
	type magnetResult struct {
		match      javdb.ImageSearchMatch
		magnets    []map[string]any
		err        error
		errorStage string
		errorCode  string
	}
	results := make([]magnetResult, len(result.Matches))
	const workers = 4
	jobs := make(chan int)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := range jobs {
				match := result.Matches[i]
				if match.Error != nil {
					results[i] = magnetResult{
						match:      match,
						err:        errors.New(match.Error.Message),
						errorStage: match.Error.Stage,
						errorCode:  match.Error.Code,
					}
					continue
				}
				if match.MovieID == "" {
					results[i] = magnetResult{match: match, err: errors.New("no movie id"), errorStage: "link", errorCode: "resolve"}
					continue
				}
				mags, err := setup.Client.MovieMagnets(ctx, match.MovieID)
				if err != nil {
					results[i] = magnetResult{match: match, err: err, errorStage: "magnets", errorCode: "fetch"}
					continue
				}
				mags = javdb.FilterMagnets(mags, cnsub, hd, minMiB)
				mags = javdb.RankMagnets(mags, magnetsCount)
				results[i] = magnetResult{match: match, magnets: mags}
			}
		}()
	}
	for i := range result.Matches {
		select {
		case jobs <- i:
		case <-ctx.Done():
			close(jobs)
			wait.Wait()
			return ctx.Err()
		}
	}
	close(jobs)
	wait.Wait()

	failures := 0
	switch mode {
	case record.OutputJSON:
		matchesOut := make([]map[string]any, 0, len(results))
		for _, r := range results {
			entry := map[string]any{
				"video_code": r.match.Candidate.VideoCode,
				"movie_id":   r.match.MovieID,
				"magnets":    r.magnets,
			}
			if r.err != nil {
				failures++
				entry["error"] = r.err.Error()
			}
			matchesOut = append(matchesOut, entry)
		}
		if err := writeJSON(streams.Out, map[string]any{"reverse_search": result.ReverseSearch, "matches": matchesOut}); err != nil {
			return err
		}
	case record.OutputNDJSON:
		writer := record.NewWriter(streams.Out, record.OutputNDJSON)
		for _, r := range results {
			ref := r.match.Candidate.VideoCode
			id := r.match.MovieID
			if r.err != nil {
				stage, code := r.errorStage, r.errorCode
				if stage == "" {
					stage = "magnets"
				}
				if code == "" {
					code = "fetch"
				}
				if err := writer.Write(record.ErrorEnvelope(record.New(record.KindMagnet, ref, id), "search", stage, code, r.err.Error())); err != nil {
					return err
				}
				failures++
				continue
			}
			envelope := record.New(record.KindMagnet, ref, id).
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
				continue
			}
			for _, m := range r.magnets {
				uri := javdb.MagnetURI(m)
				if uri != "" {
					if _, err := fmt.Fprintln(streams.Out, uri); err != nil {
						return err
					}
				}
			}
		}
	}
	if failures > 0 {
		var causes []error
		for _, r := range results {
			if r.err != nil {
				causes = append(causes, r.err)
				if err := streams.Failure(r.err); err != nil {
					return err
				}
			}
		}
		return &logging.Reported{Failed: failures, Total: len(results), Cause: errors.Join(causes...)}
	}
	return nil
}
