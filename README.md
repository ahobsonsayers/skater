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

The scrape traverses down via `NextPageRequests` (like a call stack going down) and back up via `CollectPageResults` (like returns bubbling up). Result data flows in two directions:

- **Up-flow:** every page's result data — including explicit nils — bubbles up through its parents. A parent can observe the data of the pages it spawned by implementing `CollectPageResults(childResults []any)`; collecting does not consume the results, they keep bubbling up.
- **Final results:** the same result data enters the final results unless the page implements `ShouldEmit()` returning `false`.

What result data a page carries is controlled by the `Result` method:

- **No `Result` method:** the page struct itself is the result data. Alternatively, implement a method that simply returns the receiver.
- **`Result` returns data:** that data is the result data instead of the page.
- **`Result` returns `nil`:** the nil itself is the result data — an explicit choice, passed up to collectors and emitted into the final results unless `ShouldEmit` gates it. Combine with `ShouldEmit` returning `false` for a page that only enriches its children.

Working examples live in [`examples/`](examples/): 
 - [`aggregation`](examples/aggregation) (parent collects child results, `ShouldEmit` silences children)
 - [`injection`](examples/injection) (parent carries no data, children emit)
 - [`crawl`](examples/crawl) (parent and children all emit)
