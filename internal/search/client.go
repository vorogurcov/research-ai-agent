package search

import "context"

type Client interface {
	Search(ctx context.Context, query string, page int) (string, error)
}
