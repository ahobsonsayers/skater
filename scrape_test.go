package skater

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const (
	book1URL    = "https://example.com/books/1"
	book2URL    = "https://example.com/books/2"
	categoryURL = "https://example.com/category"
	chain1URL   = "https://example.com/chain/1"
	chain2URL   = "https://example.com/chain/2"
	chain3URL   = "https://example.com/chain/3"
)

const (
	book1HTML    = "<html><h1>Book One</h1><p class='price'>£5.00</p></html>"
	book2HTML    = "<html><h1>Book Two</h1><p class='price'>£10.00</p></html>"
	categoryHTML = "<html><h1>Travel</h1>" +
		"<a class='book-link' href='" + book1URL + "'>Book 1</a>" +
		"<a class='book-link' href='" + book2URL + "'>Book 2</a>" +
		"</html>"
	chain1HTML = "<html><a id='next' href='" + chain2URL + "'>Next</a></html>"
	chain2HTML = "<html><a id='next' href='" + chain3URL + "'>Next</a></html>"
	chain3HTML = "<html></html>"
)

type fakeFetcher struct {
	pages   map[string]string
	fetches int
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	f.fetches++

	html, ok := f.pages[url]
	if !ok {
		return nil, fmt.Errorf("no page for url '%s'", url)
	}

	return io.NopCloser(strings.NewReader(html)), nil
}

func newTestScraper(t *testing.T, pages map[string]string, maxDepth int) *Scraper {
	t.Helper()

	fetcher := &fakeFetcher{pages: pages}
	scraper, err := NewScraper(Config{
		Fetcher:  fetcher,
		MaxDepth: maxDepth,
	})
	if err != nil {
		t.Fatalf("new scraper: %s", err)
	}

	return scraper
}

type testCategory struct {
	Name  string
	Books int
}

// testCategoryPage is a parent page: parses book links, crawls into children,
// collects their results and emits a summary. Parsed using goquery.
type testCategoryPage struct {
	Name      string
	BookURLs  []string
	Collected []any
}

func (p *testCategoryPage) Parse(r io.Reader) error {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return fmt.Errorf("parse document: %w", err)
	}

	p.Name = document.Find("h1").First().Text()

	document.Find("a.book-link").Each(
		func(_ int, selection *goquery.Selection) {
			bookURL, ok := selection.Attr("href")
			if !ok {
				return
			}

			p.BookURLs = append(p.BookURLs, bookURL)
		},
	)

	return nil
}

func (p *testCategoryPage) NextPageRequests() []PageRequest {
	requests := make([]PageRequest, 0, len(p.BookURLs))
	for _, bookURL := range p.BookURLs {
		requests = append(
			requests,
			PageRequest{
				URL:  bookURL,
				Page: &testBookPage{},
			},
		)
	}

	return requests
}

func (p *testCategoryPage) CollectPageResults(childResults []any) {
	p.Collected = childResults
}

func (p *testCategoryPage) Result() any {
	return testCategory{
		Name:  p.Name,
		Books: len(p.BookURLs),
	}
}

// testBookPage is a leaf page parsed by pagser using struct tags.
type testBookPage struct {
	Title string `pagser:"h1"`
	Price string `pagser:"p.price"`
}

// testHiddenPage suppresses its result by implementing PageShouldEmit.
type testHiddenPage struct {
	Title string `pagser:"h1"`
}

func (*testHiddenPage) ShouldEmit() bool {
	return false
}

// testChainPage links to the next page in a chain to test MaxDepth.
type testChainPage struct {
	NextURL string
}

func (p *testChainPage) Parse(r io.Reader) error {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return fmt.Errorf("parse document: %w", err)
	}

	nextURL, ok := document.Find("a#next").First().Attr("href")
	if ok {
		p.NextURL = nextURL
	}

	return nil
}

func (p *testChainPage) NextPageRequests() []PageRequest {
	if p.NextURL == "" {
		return nil
	}

	return []PageRequest{
		{
			URL:  p.NextURL,
			Page: &testChainPage{},
		},
	}
}

func TestScrapePages(t *testing.T) {
	scraper := newTestScraper(
		t,
		map[string]string{
			categoryURL: categoryHTML,
			book1URL:    book1HTML,
			book2URL:    book2HTML,
		},
		5,
	)

	parentPage := &testCategoryPage{}
	results, err := scraper.Scrape(
		context.Background(),
		[]PageRequest{{URL: categoryURL, Page: parentPage}},
	)
	if err != nil {
		t.Fatalf("scrape: %s", err)
	}

	// Children emitted before the parent, in request order
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	category, ok := results[2].(testCategory)
	if !ok {
		t.Fatalf("expected parent result to be testCategory, got %T", results[2])
	}

	if category.Name != "Travel" || category.Books != 2 {
		t.Fatalf("expected category {Travel 2}, got %+v", category)
	}

	books := GetResults[*testBookPage](results)
	if len(books) != 2 || books[0].Title != "Book One" || books[1].Title != "Book Two" {
		t.Fatalf("expected two books in order, got %+v", books)
	}

	// Collected results match the emitted children
	if !reflect.DeepEqual(parentPage.Collected, []any{books[0], books[1]}) {
		t.Fatalf("expected collected children %+v, got %+v", books, parentPage.Collected)
	}
}

func TestScrapeShouldEmit(t *testing.T) {
	scraper := newTestScraper(
		t,
		map[string]string{
			book1URL: book1HTML,
		},
		1,
	)

	results, err := scraper.Scrape(
		context.Background(),
		[]PageRequest{{URL: book1URL, Page: &testHiddenPage{}}},
	)
	if err != nil {
		t.Fatalf("scrape: %s", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected no emitted results, got %+v", results)
	}
}

func TestScrapeMaxDepth(t *testing.T) {
	fetcher := &fakeFetcher{
		pages: map[string]string{
			chain1URL: chain1HTML,
			chain2URL: chain2HTML,
			chain3URL: chain3HTML,
		},
	}

	scraper, err := NewScraper(Config{
		Fetcher:  fetcher,
		MaxDepth: 2,
	})
	if err != nil {
		t.Fatalf("new scraper: %s", err)
	}

	results, err := scraper.Scrape(
		context.Background(),
		[]PageRequest{{URL: chain1URL, Page: &testChainPage{}}},
	)
	if err != nil {
		t.Fatalf("scrape: %s", err)
	}

	// Third page is beyond MaxDepth so never fetched
	if len(results) != 2 {
		t.Fatalf("expected 2 results at MaxDepth 2, got %d", len(results))
	}

	if fetcher.fetches != 2 {
		t.Fatalf("expected 2 fetches, got %d", fetcher.fetches)
	}
}

func TestScrapeErrors(t *testing.T) {
	tests := []struct {
		name     string
		pages    map[string]string
		wantErr  string
		requests []PageRequest
	}{
		{
			name:    "non pointer page",
			wantErr: "page must be a non-nil pointer",
			requests: []PageRequest{
				{URL: book1URL, Page: testBookPage{}},
			},
		},
		{
			name:    "fetch failure",
			wantErr: "failed to fetch content from url",
			requests: []PageRequest{
				{URL: "https://example.com/missing", Page: &testBookPage{}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scraper := newTestScraper(t, test.pages, 1)

			_, err := scraper.Scrape(context.Background(), test.requests)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing '%s', got %v", test.wantErr, err)
			}
		})
	}
}

func TestNewScraper(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		mustErr      bool
		wantMaxDepth int
	}{
		{
			name:    "negative max depth",
			config:  Config{MaxDepth: -1},
			mustErr: true,
		},
		{
			name:         "default max depth",
			config:       Config{},
			wantMaxDepth: 10,
		},
		{
			name:         "explicit max depth",
			config:       Config{MaxDepth: 3},
			wantMaxDepth: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := NewScraper(test.config)

			if test.mustErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if scraper.config.MaxDepth != test.wantMaxDepth {
				t.Fatalf("expected MaxDepth %d, got %d", test.wantMaxDepth, scraper.config.MaxDepth)
			}

			if scraper.config.Fetcher == nil {
				t.Fatal("expected default fetcher to be set")
			}
		})
	}
}
