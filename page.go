package skater

import (
	"io"
)

// PageRequest holds information about a page to scrape.
type PageRequest struct {
	// URL to fetch
	URL string

	// Page instance to parse into.
	// Must be a pointer to struct.
	Page any
}

// PageParser allows custom html parsing.
// If not implemented, parsing is done using pagser and struct tags.
type PageParser interface {
	// Parse reads html from the reader and populates the page struct.
	Parse(r io.Reader) error
}

// PageResult allows returning custom result data from a page struct.
// If not implemented, the page struct itself is used as result data.
//
// Returning nil is an explicit choice: the nil itself becomes the page's
// result data (e.g. a parent that only enriches its children). Combine
// with ShouldEmit returning false to keep it out of the final results.
type PageResult interface {
	// Result returns the scraped data from the page.
	Result() any
}

// PageShouldEmit allows controlling whether a page's result data is
// emitted into the final results.
// If not implemented, result data is emitted by default.
type PageShouldEmit interface {
	// ShouldEmit returns whether the page's result data should be
	// emitted into the final results.
	ShouldEmit() bool
}

// PageNextRequests allows defining next pages to scrape.
// If not implemented, or the method returns nil or an empty slice, the
// page is treated as a leaf page and no further scraping is done.
type PageNextRequests interface {
	// NextPageRequests returns the next pages to scrape.
	NextPageRequests() []PageRequest
}

// PageCollectResults allows collecting the result data of child pages.
// If implemented, it is called once the pages spawned by NextPageRequests
// have completed and produced results. Collecting does not consume the
// results - they still bubble up to the next parent.
type PageCollectResults interface {
	// CollectPageResults receives the result data of the child pages
	// spawned by NextPageRequests, in request order.
	CollectPageResults(childResults []any)
}
