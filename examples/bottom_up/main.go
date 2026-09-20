// Bottom-Up example: Parent aggregates data from child pages.
//
// Run: go run ./examples/bottom_up/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/ahobsonsayers/skater"
)

type SimpleFetcher struct{}

func (f *SimpleFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

type CategorySummary struct {
	Category   string
	BookCount  int
	TotalPrice float64
	Books      []string
}

func (c *CategoryBottomUp) Result() any {
	var total float64
	titles := make([]string, 0, len(c.Books))
	for _, b := range c.Books {
		if b != nil && b.Title != "" {
			titles = append(titles, b.Title)
			priceStr := fmt.Sprintf("%s", b.Price)
			priceStr = priceStr[1:]
			fmt.Sscanf(priceStr, "%f", &total)
		}
	}
	return CategorySummary{
		Category:   c.Category,
		BookCount:  len(titles),
		TotalPrice: total,
		Books:      titles,
	}
}

type BookDetailPage struct {
	Title string `pagser:"div.product_main h1"`
	Price string `pagser:"p.price_color"`
}

func (b *BookDetailPage) Parse(r io.Reader) error {
	return skater.DefaultParser.ParseReader(b, r)
}

func (b *BookDetailPage) NextRequests() []skater.PageRequest {
	return nil
}

func (b *BookDetailPage) Result() any {
	return b
}

type CategoryBottomUp struct {
	BookURLs []string
	Category string
	Books    []*BookDetailPage
}

func (c *CategoryBottomUp) Parse(r io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}
	c.Category = doc.Find("div.page-header h1").First().Text()
	baseURL := "http://books.toscrape.com/catalogue/"
	doc.Find("h3 a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if ok {
			url := href
			for len(url) > 2 && url[:3] == "../" {
				url = url[3:]
			}
			url = baseURL + url
			c.BookURLs = append(c.BookURLs, url)
		}
	})
	return nil
}

func (c *CategoryBottomUp) NextRequests() []skater.PageRequest {
	c.Books = make([]*BookDetailPage, len(c.BookURLs))
	var reqs []skater.PageRequest
	for i, url := range c.BookURLs {
		book := &BookDetailPage{}
		c.Books[i] = book
		reqs = append(reqs, skater.PageRequest{URL: url, Page: book})
	}
	return reqs
}

func main() {
	fetcher := &SimpleFetcher{}
	category := &CategoryBottomUp{}

	scraper, err := skater.NewScraper(skater.Config{Fetcher: fetcher})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	results, err := scraper.Scrape(context.Background(), []skater.PageRequest{
		{URL: "http://books.toscrape.com/catalogue/category/books/travel_2/index.html", Page: category},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	for _, r := range results {
		if summary, ok := r.(CategorySummary); ok {
			fmt.Printf("Category: %s\n", summary.Category)
			fmt.Printf("Books found: %d\n", summary.BookCount)
			fmt.Printf("Total price: £%.2f\n", summary.TotalPrice)
			fmt.Println("Books:")
			for _, title := range summary.Books {
				fmt.Printf("  - %s\n", title)
			}
		}
	}
}