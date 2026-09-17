package javdb

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
)

// MovieAssetMetadata 是通过读取资产媒体本身得到的 best-effort 元数据。
// Duration 的单位是秒，仅用于 preview video；零值表示未知。
type MovieAssetMetadata struct {
	Width    int `json:"width,omitempty"`
	Height   int `json:"height,omitempty"`
	Duration int `json:"duration,omitempty"`
}

// MovieAssetInfo 把稳定的下载身份与可选 probe metadata 分离。
type MovieAssetInfo struct {
	Asset    MovieAsset         `json:"asset"`
	Metadata MovieAssetMetadata `json:"metadata"`
}

// MovieAssetProbeOptions 控制媒体探测。
type MovieAssetProbeOptions struct {
	// Concurrency 为 0 时使用 settings.DefaultAssetProbeConcurrency；负数无效。
	Concurrency int
}

type movieAssetProbeKey struct {
	Type string
	URL  string
}

type movieAssetProbeJob struct {
	asset   MovieAsset
	indexes []int
}

// ProbeMovieAssets 以固定 worker pool 探测媒体元数据，保持输入顺序并按 Type+URL 去重。
// 单项媒体错误保留零值并继续；父 context 取消或截止时间会终止整个调用。
func (c *Client) ProbeMovieAssets(ctx context.Context, assets []MovieAsset, options MovieAssetProbeOptions) ([]MovieAssetInfo, error) {
	concurrency := options.Concurrency
	if concurrency < 0 {
		return nil, fmt.Errorf("asset probe concurrency must be non-negative, got %d", concurrency)
	}
	if concurrency == 0 {
		concurrency = settings.DefaultAssetProbeConcurrency
	}
	// 已取消的调用无需先构建完整 job/index 结构。
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	infos := make([]MovieAssetInfo, len(assets))
	jobs := make([]movieAssetProbeJob, 0, len(assets))
	jobIndexes := make(map[movieAssetProbeKey]int, len(assets))
	for index, asset := range assets {
		infos[index].Asset = asset
		if asset.Type != assetTypeImage && asset.Type != assetTypeVideo {
			continue
		}
		key := movieAssetProbeKey{Type: asset.Type, URL: asset.URL}
		if jobIndex, ok := jobIndexes[key]; ok {
			jobs[jobIndex].indexes = append(jobs[jobIndex].indexes, index)
			continue
		}
		jobIndexes[key] = len(jobs)
		jobs = append(jobs, movieAssetProbeJob{asset: asset, indexes: []int{index}})
	}
	if len(jobs) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return infos, nil
	}

	jobQueue := make(chan movieAssetProbeJob, len(jobs))
	for _, job := range jobs {
		jobQueue <- job
	}
	close(jobQueue)

	var workers sync.WaitGroup
	workerCount := min(concurrency, len(jobs))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobQueue:
					if !ok {
						return
					}
					metadata, err := c.probeMovieAsset(ctx, job.asset)
					if err != nil {
						// 单项失败始终 best-effort：媒体 transport 自身超时也会返回
						// context.DeadlineExceeded，此时父 ctx 仍有效。
						// 父 context 失败已由下面的 ctx.Err() 统一返回。
						continue
					}
					for _, index := range job.indexes {
						infos[index].Metadata = metadata
					}
				}
			}
		}()
	}
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return infos, nil
}

// normalizeProbeDuration 把 probe 得到的浮点秒数转换成公开的整数秒。
// NaN/Inf、≤ 0、以及超出 int 可表示范围的数值一律视为 duration unknown，
// 避免 float→int 的未定义降级产生虚假时长；>0 但不足 1 秒的片段向上取整为 1。
func normalizeProbeDuration(seconds float64) int {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return 0
	}
	rounded := math.Round(seconds)
	// float64(math.MaxInt) 会向上舍入到 MaxInt+1，所以用 >= 把整个边界带
	// 都归入 unknown，避免 int(rounded) 在边界上溢出成负值。
	if rounded >= float64(math.MaxInt) {
		return 0
	}
	duration := int(rounded)
	if duration < 1 {
		return 1
	}
	return duration
}

func (c *Client) probeMovieAsset(ctx context.Context, asset MovieAsset) (MovieAssetMetadata, error) {
	var (
		width       int
		height      int
		durationRaw float64
	)
	switch asset.Type {
	case assetTypeImage:
		metadata, err := c.api.ProbeImageMetadata(ctx, asset.URL)
		if err != nil {
			return MovieAssetMetadata{}, err
		}
		width, height = metadata.Width, metadata.Height
	case assetTypeVideo:
		metadata, err := c.api.ProbeHLSMetadata(ctx, asset.URL)
		if err != nil {
			return MovieAssetMetadata{}, err
		}
		width, height, durationRaw = metadata.Width, metadata.Height, metadata.DurationSeconds
	default:
		return MovieAssetMetadata{}, nil
	}
	return MovieAssetMetadata{Width: width, Height: height, Duration: normalizeProbeDuration(durationRaw)}, nil
}
