package appapi

import "encoding/json"

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
// Requires bearer token.
func (c *Client) MovieMagnets(movieID string) ([]map[string]any, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/"+movieID+"/magnets", nil, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data["magnets"]), nil
}
