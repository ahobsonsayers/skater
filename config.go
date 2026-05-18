package skater

import "fmt"

// Config is the configuration of a Scraper.
type Config struct {
	// Fetcher is used to fetch html content.
	// Defaults to a HTTPFetcher that uses the default *http.Client.
	Fetcher Fetcher

	// MaxDepth limits how many levels of pages to follow.
	// A value of 0 or less is invalid.
	// Defaults to 10.
	MaxDepth int
}

func (c *Config) defaults() {
	if c.Fetcher == nil {
		c.Fetcher = NewHTTPFetcher()
	}
	if c.MaxDepth == 0 {
		c.MaxDepth = 10
	}
}

func (c *Config) validate() error {
	if c.MaxDepth <= 0 {
		return fmt.Errorf("max depth must be greater than 0, got %d", c.MaxDepth)
	}
	return nil
}
