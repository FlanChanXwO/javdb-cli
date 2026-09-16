package media

import (
	"bytes"
	"context"
	"io"
)

// byteFetch 让历史 fixture 继续以 []byte 描述资源,同时明确模拟新的流式 FetchContext。
func byteFetch(fetch func(context.Context, string) ([]byte, error)) FetchContext {
	return func(ctx context.Context, uri string) (io.ReadCloser, error) {
		data, err := fetch(ctx, uri)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}
