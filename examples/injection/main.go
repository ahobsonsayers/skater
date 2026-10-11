// Injection example: Parent page injects data into child pages which emit results.
//
// The parent pre-fills child pages with known data. Each child emits
// itself as the result. The parent emits no result data.
//
// Run: go run ./examples/injection/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ahobsonsayers/skater"
)

type Fetcher struct{}

func (*Fetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

// BookCategoryPage holds data parsed from the book category web page
//
// It is not the type of the final result we want. See BookPage
type BookCategoryPage struct {
	Category string
	BookURLs []string
}

func (c *BookCategoryPage) Parse(r io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}

	// Parse category name
	c.Category = doc.Find("div.page-header h1").First().Text()

	// Parse URLs of books in category
	baseURL := "https://books.toscrape.com/catalogue/"
	doc.Find("h3 a").Each(
		func(_ int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if !ok {
				return
			}

			for strings.HasPrefix(href, "../") {
				href = strings.TrimPrefix(href, "../")
			}

			bookURL := baseURL + href
			c.BookURLs = append(c.BookURLs, bookURL)
		},
	)
	return nil
}

// NextPageRequests returns requests to crawl into after this page.
// If not defined, crawling ends at this page.
func (c *BookCategoryPage) NextPageRequests() []skater.PageRequest {
	requests := make([]skater.PageRequest, len(c.BookURLs))
	for i, bookURL := range c.BookURLs {
		requests[i] = skater.PageRequest{
			URL: bookURL,
			// Inject parent data into the child page up front
			Page: &BookPage{
				Category: c.Category,
			},
		}
	}

	return requests
}

// ShouldEmit returns false so category data is never emitted into the final results.
// The parent simply enriches the child page data.
func (*BookCategoryPage) ShouldEmit() bool {
	return false
}

// BookPage is the type of the final result we want.
//
// As it does not have a Parse method, pagser will be used to parse.
// As it does not have a NextPageRequests method, crawling will end here.
// As it does not have a Result method, the type parsed into is
// returned as the result as is. Every book page is therefore emitted
// as a result.
//
// The parent injects the category into each child page before it is
// scraped. The child page parses the remaining data itself.
// The ShouldEmit method returns false, so category data is not emitted as a result.
// Category data is use to inject data into the child page instead.
type BookPage struct {
	Title    string `pagser:"div.product_main h1"`
	Price    string `pagser:"p.price_color"`
	Category string
}

func main() {
	fetcher := &Fetcher{}
	config := skater.Config{Fetcher: fetcher}

	scraper, err := skater.NewScraper(config)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	initialRequest := skater.PageRequest{
		URL:  "https://books.toscrape.com/catalogue/category/books/travel_2/index.html",
		Page: &BookCategoryPage{},
	}

	results, err := scraper.Scrape(
		context.Background(),
		[]skater.PageRequest{initialRequest},
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Results contain only *BookPage pages
	bookPages := skater.GetResults[*BookPage](results)
	for _, bookPage := range bookPages {
		fmt.Printf("Category: %s\n", bookPage.Category)
		fmt.Printf("Title: %s\n", bookPage.Title)
		fmt.Printf("Price: %s\n", bookPage.Price)
	}
}
