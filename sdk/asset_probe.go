package javdb

import (
	"context"
	"errors"
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
// 单项媒体错误保留零值并继续；context 取消或截止时间会终止整个调用。
func (c *Client) ProbeMovieAssets(ctx context.Context, assets []MovieAsset, options MovieAssetProbeOptions) ([]MovieAssetInfo, error) {
	concurrency := options.Concurrency
	if concurrency < 0 {
		return nil, fmt.Errorf("asset probe concurrency must be non-negative, got %d", concurrency)
	}
	if concurrency == 0 {
		concurrency = settings.DefaultAssetProbeConcurrency
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

	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	jobQueue := make(chan movieAssetProbeJob, len(jobs))
	for _, job := range jobs {
		jobQueue <- job
	}
	close(jobQueue)

	var (
		workers    sync.WaitGroup
		contextErr error
		errorOnce  sync.Once
	)
	workerCount := min(concurrency, len(jobs))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-workerContext.Done():
					return
				case job, ok := <-jobQueue:
					if !ok {
						return
					}
					metadata, err := c.probeMovieAsset(workerContext, job.asset)
					if err != nil {
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							errorOnce.Do(func() {
								contextErr = err
								cancel()
							})
						}
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
	if contextErr != nil {
		return nil, contextErr
	}
	return infos, nil
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
	duration := int(math.Round(durationRaw))
	if durationRaw > 0 && duration < 1 {
		duration = 1
	}
	return MovieAssetMetadata{Width: width, Height: height, Duration: duration}, nil
}
