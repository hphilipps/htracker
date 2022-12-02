# HTracker - track website updates

## Use Cases

1. Add/remove URL from scrape list
2. Configure filters per URL
3. Configure scrape intervals per URL
4. Provide web interface to show last updates of URL(s)
5. Provide RSS feed for streaming changes
6. Register/unregister email for notifications on changes on URL

## Design

1. spider - reaches out to set of sites
2. feed - serves feed of changes for subscribers
3. push news - push news out to subscribers

## Scrape

1. NewScraper + opts per set of URLs
2. Scraper.Start()
3. NewExporter() -> register results (date, txt, checksum, diff)

## Find Updates