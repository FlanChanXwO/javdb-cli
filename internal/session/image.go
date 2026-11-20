package session

import (
	"context"

	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// SearchByImage uses the public SDK linker with authenticated lookups, so a rejected
// JavDB operation is replayed without uploading the image to the provider again.
func (c *Client) SearchByImage(ctx context.Context, request javdb.ReverseSearchRequest, opts javdb.ImageSearchOptions) (javdb.ImageSearchResult, error) {
	response, err := c.Client.ReverseSearch(ctx, request)
	if err != nil {
		return javdb.ImageSearchResult{}, err
	}
	return javdb.LinkImageCandidates(ctx, c, response, opts)
}
