# skater

> He was a scraper boy, she said see you later boy

**skater** is a stupidly simple scraping framework that simply gets out of your way.

There are plenty of scraping libraries packed with features for crawling massive amounts of unstructured data. However, their open-ended, event-driven approach relies heavily on callbacks and global state. This often leaves you writing extensive post-processing logic just to wrangle the scraped data into the shape your application actually needs.

**skater** takes a different approach. It is built for targeted extraction—when you already know the page structure and simply want to pull specific data directly into your exact formats. It shines when you need to scrape data synchronously as part of your application's core logic, rather than running heavy, background crawls.

Under the hood, it uses the wonderful `pagser` library to parse HTML using struct tags—exactly how Go handles JSON or YAML. You just define your struct, tag it, and let the framework do the rest.

### Features

- **Struct-Driven Parsing:** Define your scraping rules and routing directly in your domain structs. No messy global state or `OnHTML` hooks.
- **Type-Safe Extraction:** Uses Go generics to yield clean, fully-typed data. Say goodbye to `interface{}` casting.
- **Smart Data Enrichment:** Easily pass context down to child pages, or aggregate child data back up to a parent object using standard Go pointers.
- **Pluggable Fetcher:** A lean interface makes it trivial to swap the default HTTP client for a proxy rotator or headless browser (like Playwright).

### Results

Each page controls what it emits into the final results by implementing (or not) the `Result` method:

- **No `Result` method:** the page struct itself is returned as the result. Alternatively, implement a method that simply returns the receiver.
- **`Result` returns data:** that data is returned as the result instead of the page.
- **`Result` returns `nil`:** no result data is emitted at all — use this for pages that only exist to enrich other pages (e.g. a parent that aggregates its children's data).

Working examples of all three modes live in [`examples/`](examples/): [`aggregation`](examples/aggregation), [`injection`](examples/injection), and [`crawl`](examples/crawl).
