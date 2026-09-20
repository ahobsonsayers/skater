// Bottom-Up example: Parent aggregates data from child pages.
//
// The category page (parent) creates a page request for each book,
// keeping a reference to every child page it creates. As child pages
// complete, the shared structs are filled in, and the parent reads
// them back in Result to build the final BookCategory.
//
// Run: go run ./examples/bottom_up/main.go
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

func (f *Fetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
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
	Books    []*BookDetails
}

func (c *BookCategoryPage) ToBookCategory() BookCategory {
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

	return BookCategory{
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
	baseURL := "http://books.toscrape.com/catalogue/"
	doc.Find("h3 a").Each(
		func(i int, s *goquery.Selection) {
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

// NextRequests returns requests to crawl into after this page.
// If not defined, crawling ends at this page.
func (c *BookCategoryPage) NextRequests() []skater.PageRequest {
	// The same book pointers are shared with the requests below
	// skater fills them in as each child page completes.
	c.Books = make([]*BookDetails, len(c.BookURLs))

	requests := make([]skater.PageRequest, len(c.BookURLs))
	for i, bookURL := range c.BookURLs {
		book := &BookDetails{}
		requests[i] = skater.PageRequest{
			URL:  bookURL,
			Page: book,
		}
		c.Books[i] = book
	}

	return requests
}

// Result returns the final result of the scraping, which can be different to the page type.
// If not defined, the type parsed into is returned as the result as is.
func (c *BookCategoryPage) Result() any {
	return c.ToBookCategory()
}

// BookDetails holds data parsed from the book detail web page.
// It is also the type of the final result we want.
//
// As it does not have a Parse method, pagser will be used to parse.
//
// As it does not have a NextRequests method, crawling will end on this.
//
// Its Result method returns nil, so book pages are not emitted
type BookDetails struct {
	Title string `pagser:"div.product_main h1"`
	Price string `pagser:"p.price_color"`
}

// Result returns nil so the book page emits nothing into the results.
// / The parent aggregates child data itself, so only the category page emits a result.
func (b *BookDetails) Result() any {
	return nil
}

func main() {
	fetcher := &Fetcher{}
	config := skater.Config{Fetcher: fetcher}

	scraper, err := skater.NewScraper(config)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	initialRequest := skater.PageRequest{
		URL:  "http://books.toscrape.com/catalogue/category/books/travel_2/index.html",
		Page: &BookCategoryPage{},
	}

	results, err := scraper.Scrape(
		context.Background(),
		[]skater.PageRequest{initialRequest},
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Results contain only BookCategory pages
	for _, result := range results {
		bookCategory := result.(BookCategory)
		fmt.Printf("Category: %s\n", bookCategory.Category)
		fmt.Printf("Books found: %d\n", bookCategory.BookCount)
		fmt.Printf("Total price: £%.2f\n", bookCategory.TotalPrice)
		fmt.Println("Books:")
		for _, title := range bookCategory.BookTitles {
			fmt.Printf("  - %s\n", title)
		}
	}
}
