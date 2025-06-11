package tests

import (
	"rss-reader/internal/feed"
	"testing"
)

func TestFeedParsing(t *testing.T) {
	err := feed.ParseFeed("https://feeds.lifeworq.com/us/c8cc74ef-c4c8-4189-bdca-8ee2fbe4fb3e.xml")
	if err != nil {
		t.Fatalf("Failed to parse feed: %v", err)
	}
}
