package skater

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/imroc/req/v3"
)

// Fetcher is used to fetch html content.
type Fetcher interface {
	// Fetcher fetches html content from the url given.
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// HTTPFetcher is a Fetcher implementation using an net/http Client.
type HTTPFetcher struct {
	client *http.Client
}

// Client returns the underlying *http.Client for modification if required.
func (f *HTTPFetcher) Client() *http.Client {
	return f.client
}

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// NewHTTPFetcher creates a new HTTPFetcher with a default *http.Client.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{},
	}
}

// WrapHTTPClient wraps an existing *http.Client as a Fetcher.
func WrapHTTPClient(client *http.Client) *HTTPFetcher {
	return &HTTPFetcher{client: client}
}

// ReqFetcher implements the Fetcher interface using an imroc/req Client.
type ReqFetcher struct {
	client *req.Client
}

// Client returns the underlying *req.Client for modification if required.
func (f *ReqFetcher) Client() *req.Client {
	return f.client
}

func (f *ReqFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	resp, err := f.client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch url '%s': %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// NewReqFetcher creates a new ReqFetcher with a default *req.Client.
func NewReqFetcher() *ReqFetcher {
	return &ReqFetcher{
		client: req.C(),
	}
}

// WrapReqClient wraps an existing *req.Client as a Fetcher.
func WrapReqClient(client *req.Client) *ReqFetcher {
	return &ReqFetcher{client: client}
}
