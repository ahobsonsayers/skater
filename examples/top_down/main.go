// Top-Down example: Parent injects data into child pages.
//
// Run: go run ./examples/top_down/main.go
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

type CategoryTopDown struct {
	BookURLs []string
	Category string
	Books    []struct {
		URL   string
		Title string
	}
}

func (c *CategoryTopDown) Parse(r io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}
	c.Category = doc.Find("div.page-header h1").First().Text()
	baseURL := "http://books.toscrape.com/catalogue/"
	doc.Find("h3 a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		title, titleOk := s.Attr("title")
		if ok {
			url := href
			for len(url) > 2 && url[:3] == "../" {
				url = url[3:]
			}
			url = baseURL + url
			c.BookURLs = append(c.BookURLs, url)
			if titleOk {
				c.Books = append(c.Books, struct {
					URL   string
					Title string
				}{URL: url, Title: title})
			}
		}
	})
	return nil
}

func (c *CategoryTopDown) NextRequests() []skater.PageRequest {
	var reqs []skater.PageRequest
	for _, bookInfo := range c.Books {
		reqs = append(reqs, skater.PageRequest{
			URL: bookInfo.URL,
			Page: &BookPageTopDown{
				Category: c.Category,
				Title:    bookInfo.Title,
			},
		})
	}
	return reqs
}

func (c *CategoryTopDown) Result() any {
	return nil
}

type BookPageTopDown struct {
	Category string
	Title    string `pagser:"div.product_main h1"`
	Price    string `pagser:"p.price_color"`
}

func (b *BookPageTopDown) Parse(r io.Reader) error {
	title := b.Title
	err := skater.DefaultParser.ParseReader(b, r)
	b.Title = title
	return err
}

func (b *BookPageTopDown) NextRequests() []skater.PageRequest {
	return nil
}

func (b *BookPageTopDown) Result() any {
	return b
}

func main() {
	fetcher := &SimpleFetcher{}
	category := &CategoryTopDown{}

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

	fmt.Println("Books found:")
	for _, r := range results {
		if book, ok := r.(*BookPageTopDown); ok {
			fmt.Printf("  - [%s] %s @ %s\n", book.Category, book.Title, book.Price)
		}
	}
}