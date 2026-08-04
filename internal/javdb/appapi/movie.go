package appapi

import (
	"encoding/json"
	"strconv"
)

// MovieDetail returns the nested movie object from GET /api/v4/movies/{id}.
func (c *Client) MovieDetail(movieID string) (map[string]any, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v4/movies/"+movieID, nil, &data); err != nil {
		return nil, err
	}
	if raw, ok := data["movie"]; ok && len(raw) > 0 && string(raw) != "null" {
		var movie map[string]any
		if err := json.Unmarshal(raw, &movie); err != nil {
			return nil, err
		}
		return movie, nil
	}
	// fallback: whole data as movie-ish
	var flat map[string]any
	b, _ := json.Marshal(data)
	_ = json.Unmarshal(b, &flat)
	return flat, nil
}

// MovieMagnets returns magnets for an internal id (GET /api/v1/movies/{id}/magnets).
// Works without a bearer token; the token (when set) is forwarded as-is.
func (c *Client) MovieMagnets(movieID string) ([]map[string]any, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/"+movieID+"/magnets", nil, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data["magnets"]), nil
}

// MovieComments 获取影片评论的一页数据。它只请求调用方指定的一页，绝不自动追取后续页。
func (c *Client) MovieComments(movieID string, page, limit int) ([]map[string]any, error) {
	params := movieCommentsParams(page, limit)
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/"+movieID+"/reviews", params, &data); err != nil {
		return nil, err
	}
	return decodeMovieComments(data["reviews"]), nil
}

func movieCommentsParams(page, limit int) map[string]string {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		// 与 browse 等单页命令保持相同默认值；不会据此追加请求其他评论页。
		limit = 20
	}
	return map[string]string{
		"page":  strconv.Itoa(page),
		"limit": strconv.Itoa(limit),
	}
}

func decodeMovieComments(raw json.RawMessage) []map[string]any {
	items := decodeObjectArray(raw)
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		// 评论列表的上游数据可能带 null；向 CLI/SDK 暴露的结果只保留有效对象。
		if item != nil {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
