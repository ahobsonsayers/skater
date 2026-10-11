// Crawl example: Parent and children pages emit results.
//
// The parent emits a CategoryStats summary. Each child emits itself.
// Every page contributes a result.
//
// Run: go run ./examples/crawl/main.go
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

// BookCategory is the type of the final result we want.
//
// This is returned by the Result method of BookCategoryPage.
type BookCategory struct {
	Category string
	BookURLs int
}

// BookCategoryPage holds data parsed from the book category web page
//
// It is not the type of the final result we want. See BookCategory
type BookCategoryPage struct {
	Category string
	BookURLs []string
}

func (c *BookCategoryPage) ToBookCategory() *BookCategory {
	return &BookCategory{
		Category: c.Category,
		BookURLs: len(c.BookURLs),
	}
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
			URL:  bookURL,
			Page: &BookInfo{},
		}
	}

	return requests
}

// Result returns the final result of the scraping, which can be different to the page type.
// If not defined, the type parsed into is itself returned as the result.
func (c *BookCategoryPage) Result() any {
	// Multi-emission: the parent also emits a result.
	return c.ToBookCategory()
}

// BookInfo holds data parsed from the book detail web page.
// It is also the type of the final result we want.
//
// As it does not have a Parse method, pagser will be used to parse.
// As it does not have a NextPageRequests method, crawling will end here.
// As it does not have a Result method, this itself is returned as the result.
type BookInfo struct {
	Title string `pagser:"div.product_main h1"`
	Price string `pagser:"p.price_color"`
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

	// Results contain both *BookCategory and *BookInfo entries
	bookCategories := skater.GetResults[*BookCategory](results)
	for _, bookCategory := range bookCategories {
		fmt.Printf("[Category] %s - %d books\n", bookCategory.Category, bookCategory.BookURLs)
	}

	books := skater.GetResults[*BookInfo](results)
	for _, book := range books {
		fmt.Printf("[Book] %s @ %s\n", book.Title, book.Price)
	}
}
