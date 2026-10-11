package skater

import (
	"context"
	"fmt"
	"reflect"
	"slices"

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
	pageResults, err := s.scrapeRecursive(ctx, initialPageRequests, 0)

	results := make([]any, 0, len(pageResults))
	for _, pageResult := range pageResults {
		if pageResult.ShouldEmit {
			results = append(results, pageResult.Result)
		}
	}

	return slices.Clip(results), err
}

type pageResult struct {
	Result     any
	ShouldEmit bool
}

func (s *Scraper) scrapeRecursive(ctx context.Context, requests []PageRequest, depth int) ([]pageResult, error) {
	if depth >= s.config.MaxDepth {
		return nil, nil
	}

	var results []pageResult
	for _, request := range requests {
		// Check if context has been cancelled
		err := ctx.Err()
		if err != nil {
			return results, err
		}

		// Check the page type we will unmarshal to is a pointer
		page := request.Page
		if page == nil || reflect.TypeOf(page).Kind() != reflect.Pointer {
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
		if parser, ok := page.(PageParser); ok {
			parseErr = parser.Parse(htmlReader)
		} else {
			parseErr = DefaultParser.ParseReader(page, htmlReader)
		}
		htmlReader.Close()
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse content from url '%s': %w", request.URL, parseErr)
		}

		// Get next page requests using the NextPageRequests method if defined.
		var nextRequests []PageRequest
		if p, ok := page.(PageNextRequests); ok {
			nextRequests = p.NextPageRequests()
		}

		// Recursively scrape
		var childPageResults []pageResult
		if len(nextRequests) > 0 {
			var err error
			childPageResults, err = s.scrapeRecursive(ctx, nextRequests, depth+1)
			if err != nil {
				return nil, err
			}
		}

		// Collect child results using CollectPageResults method if defined
		if p, ok := page.(PageCollectResults); ok {
			childResults := make([]any, 0, len(childPageResults))
			for _, childResult := range childPageResults {
				childResults = append(childResults, childResult.Result)
			}

			p.CollectPageResults(childResults)
		}

		// Get result data using the Result method if defined,
		// otherwise falling back to the page as the data
		var result any
		if p, ok := page.(PageResult); ok {
			result = p.Result()
		} else {
			result = page
		}

		// Return results that should be emitted
		shouldEmit := true
		if p, ok := page.(PageShouldEmit); ok {
			shouldEmit = p.ShouldEmit()
		}

		results = append(results, childPageResults...)
		results = append(
			results,
			pageResult{
				Result:     result,
				ShouldEmit: shouldEmit,
			},
		)
	}

	return results, nil
}
