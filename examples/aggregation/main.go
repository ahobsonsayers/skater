// Aggregation example: Parent page collects the results of child pages and
// emits an aggregated result.
//
// The parent page emits a BookCategory as a result.
// This result contains information aggregated from child pages.
// Each child page emits no result data.
//
// Run: go run ./examples/aggregation/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	Category   string
	BookCount  int
	TotalPrice float64
	BookTitles []string
}

// BookCategoryPage holds data parsed from the book category web page
//
// It is not the type of the final result we want. See BookCategory
type BookCategoryPage struct {
	Category string
	BookURLs []string
	Books    []*BookInfo
}

func (c *BookCategoryPage) ToBookCategory() *BookCategory {
	var totalPrice float64
	titles := make([]string, 0, len(c.Books))
	for _, book := range c.Books {
		if book == nil || book.Title == "" {
			continue
		}

		titles = append(titles, book.Title)

		price := strings.TrimPrefix(book.Price, "£")
		value, err := strconv.ParseFloat(price, 64)
		if err != nil {
			continue
		}

		totalPrice += value
	}

	return &BookCategory{
		Category:   c.Category,
		BookTitles: titles,
		BookCount:  len(titles),
		TotalPrice: totalPrice,
	}
}

// Parse parses HTML into BookCategoryPage using goquery.
// If not defined, pagser will be used by default to parse.
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

// CollectPageResults collects the result data of the child pages spawned
// by NextPageRequests, called once all child pages have completed.
func (c *BookCategoryPage) CollectPageResults(childResults []any) {
	for _, childResult := range childResults {
		c.Books = append(c.Books, childResult.(*BookInfo))
	}
}

// Result returns the final result of the scraping, which can be different to the page type.
// If not defined, the type parsed into is returned as the result as is.
func (c *BookCategoryPage) Result() any {
	return c.ToBookCategory()
}

// BookInfo holds data parsed from the book detail web page.
//
// As it does not have a Parse method, pagser will be used to parse.
// As it does not have a NextPageRequests method, crawling will end here.
// As it does not have a Result method, the page itself is its result data.
//
// The ShouldEmit method returns false, so book data is not emitted as results.
// Book data is aggregated by the parent page instead.
type BookInfo struct {
	Title string `pagser:"div.product_main h1"`
	Price string `pagser:"p.price_color"`
}

// ShouldEmit returns false so book data is never emitted into the final results.
// The parent collects and aggregates the book data into its own result instead.
func (*BookInfo) ShouldEmit() bool {
	return false
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

	// Results contain only *BookCategory values
	bookCategories := skater.GetResults[*BookCategory](results)
	for _, bookCategory := range bookCategories {
		fmt.Printf("Category: %s\n", bookCategory.Category)
		fmt.Printf("Books found: %d\n", bookCategory.BookCount)
		fmt.Printf("Total price: £%.2f\n", bookCategory.TotalPrice)
		fmt.Println("Books:")
		for _, title := range bookCategory.BookTitles {
			fmt.Printf("  - %s\n", title)
		}
	}
}
