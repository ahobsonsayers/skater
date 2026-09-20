package skater

import (
	"context"
	"fmt"
	"reflect"

	"github.com/foolin/pagser"
)

var DefaultParser = pagser.New()

// Scraper runs the scraping process.
type Scraper struct {
	config Config
}

// NewScraper creates a new Scraper with the config given.
// Returns an error if the config is invalid.
func NewScraper(config Config) (*Scraper, error) {
	config.defaults()
	err := config.validate()
	if err != nil {
		return nil, err
	}

	return &Scraper{
		config: config,
	}, nil
}

// Scrape starts the scraper with the initial page requests given.
func (s *Scraper) Scrape(ctx context.Context, initialPageRequests []PageRequest) ([]any, error) {
	return s.scrapeRecursive(ctx, initialPageRequests, 0)
}

func (s *Scraper) scrapeRecursive(ctx context.Context, requests []PageRequest, depth int) ([]any, error) {
	if depth >= s.config.MaxDepth {
		return nil, nil
	}

	var results []any
	for _, request := range requests {
		// Check if context has been cancelled
		err := ctx.Err()
		if err != nil {
			return results, err
		}

		// Check the page type we will unmarshal to is a pointer
		page := request.Page
		if page == nil || reflect.TypeOf(page).Kind() != reflect.Ptr {
			return nil, fmt.Errorf("page must be a non-nil pointer, got '%T' for url '%s'", page, request.URL)
		}

		// Fetch html content
		htmlReader, err := s.config.Fetcher.Fetch(ctx, request.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch content from url '%s': %w", request.URL, err)
		}

		// Parse html content either using the Parse method if defined,
		// or by falling back to tag parsing using pagser
		var parseErr error
		if p, ok := page.(PageParser); ok {
			parseErr = p.Parse(htmlReader)
		} else {
			parseErr = DefaultParser.ParseReader(page, htmlReader)
		}
		htmlReader.Close()
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse content from url '%s': %w", request.URL, parseErr)
		}

		// Get next page requests using the NextRequests method if defined.
		var nextRequests []PageRequest
		if p, ok := page.(PageNextRequests); ok {
			nextRequests = p.NextRequests()
		}

		// Recursively scrape
		if len(nextRequests) > 0 {
			subResults, err := s.scrapeRecursive(ctx, nextRequests, depth+1)
			if err != nil {
				return nil, err
			}
			results = append(results, subResults...)
		}

		// Get result page data using the Result method if defined,
		// otherwise falling back to the page as the data
		var data any
		if p, ok := page.(PageResult); ok {
			data = p.Result()
		} else {
			data = page
		}

		// A nil result means this page does not emit any result
		if data != nil {
			results = append(results, data)
		}
	}

	return results, nil
}
