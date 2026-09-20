// Multi-Emission example: Both parent and children emit data.
//
// Run: go run ./examples/multi_emission/main.go
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

type CategoryMulti struct {
	BookURLs []string
	Category string
}

func (c *CategoryMulti) Parse(r io.Reader) error {
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

func (c *CategoryMulti) NextRequests() []skater.PageRequest {
	var reqs []skater.PageRequest
	for _, url := range c.BookURLs {
		reqs = append(reqs, skater.PageRequest{URL: url, Page: &BookPageMulti{}})
	}
	return reqs
}

type CategoryStats struct {
	Category string
	BookURLs int
}

func (c *CategoryMulti) Result() any {
	return CategoryStats{
		Category: c.Category,
		BookURLs: len(c.BookURLs),
	}
}

type BookPageMulti struct {
	Title string `pagser:"div.product_main h1"`
	Price string `pagser:"p.price_color"`
}

func (b *BookPageMulti) Parse(r io.Reader) error {
	return skater.DefaultParser.ParseReader(b, r)
}

func (b *BookPageMulti) NextRequests() []skater.PageRequest {
	return nil
}

func (b *BookPageMulti) Result() any {
	return b
}

func main() {
	fetcher := &SimpleFetcher{}
	category := &CategoryMulti{}

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

	fmt.Println("Results:")
	for _, r := range results {
		switch v := r.(type) {
		case CategoryStats:
			fmt.Printf("  [Category] %s - %d books\n", v.Category, v.BookURLs)
		case *BookPageMulti:
			fmt.Printf("  [Book] %s @ %s\n", v.Title, v.Price)
		}
	}
}