[![pipeline status](https://gitlab.com/henri.philipps/htracker/badges/main/pipeline.svg)](https://gitlab.com/henri.philipps/htracker/-/commits/main)
[![coverage report](https://gitlab.com/henri.philipps/htracker/badges/main/coverage.svg?job=coverage)](https://gitlab.com/henri.philipps/htracker/-/commits/main)
[![Latest Release](https://gitlab.com/henri.philipps/htracker/-/badges/release.svg)](https://gitlab.com/henri.philipps/htracker/-/releases)

# HTracker - track website updates

HTracker is WIP and not ready for usage yet.

## Use Cases

1. Add/remove URL from scrape list
2. Configure filters per URL
3. Configure scrape intervals per URL
4. Provide web interface to show last updates of URL(s)
5. Provide RSS feed for streaming changes
6. Register/unregister email for notifications on changes on URL

## Design

1. watcher - reaches out to set of sites
2. feed - serves feed of changes for subscribers
3. push news - push news out to subscribers

## Watcher

1. Frequently, go through all subscribers, generate list of sites that need to be scraped and deduplicate them
2. Scrape sites with similar filters/content types in batches
    1. Maybe: Notify notifier?

## Notifier

1. Frequently, go through all subscriptions and notify if last notification is older than notification period (deduplicate by subscriber)
2. Maybe: send notifications immediately, if triggered by watcher? 

## Scrape

1. NewScraper + opts per set of URLs
2. Scraper.Start()
3. NewExporter() -> register results (date, txt, checksum, diff)

## Find Updates