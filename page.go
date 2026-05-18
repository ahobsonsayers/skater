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
type PageResult interface {
	// Result returns the scraped data from the page.
	Result() any
}

// PageNextRequests allows defining next pages to scrape.
// If not implemented, or the method returns nil or an empty slice, the
// page is treated as a leaf page and further scraping/recursion is done.
type PageNextRequests interface {
	// NextRequests returns the next pages to scrape.
	NextRequests() []PageRequest
}
