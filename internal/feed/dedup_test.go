package feed

import (
	"testing"
	"time"
)

func TestDeduplicateArticlesBySourceIdentifier(t *testing.T) {
	articles := []Article{
		{FeedID: "feed-1", Identifier: "entry-1", Title: "First"},
		{FeedID: "feed-1", Identifier: " entry-1 ", Title: "Changed title"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 1 {
		t.Fatalf("expected 1 unique article, got %d", len(unique))
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 duplicate match, got %d", len(matches))
	}
	if matches[0].CanonicalIndex != 0 || matches[0].DuplicateIndex != 1 {
		t.Fatalf("unexpected match: %+v", matches[0])
	}
	if matches[0].Reason != DuplicateBySourceIdentifier {
		t.Fatalf("unexpected reason %q", matches[0].Reason)
	}
}

func TestDeduplicateArticlesDoesNotCrossSourceScope(t *testing.T) {
	articles := []Article{
		{FeedID: "feed-1", Identifier: "entry-1", URL: "https://example.test/post"},
		{FeedID: "feed-2", Identifier: "entry-1", URL: "https://example.test/post"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 2 || len(matches) != 0 {
		t.Fatalf("cross-source articles must remain distinct: unique=%d matches=%v", len(unique), matches)
	}
}

func TestDeduplicateArticlesByNormalizedURL(t *testing.T) {
	articles := []Article{
		{
			FeedID: "feed-1",
			URL:    "HTTPS://Example.TEST:443/post?b=2&a=1#reader",
		},
		{
			FeedID: "feed-1",
			URL:    "https://example.test/post?a=1&b=2",
		},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 1 || len(matches) != 1 {
		t.Fatalf("expected URL duplicate: unique=%d matches=%v", len(unique), matches)
	}
	if matches[0].Reason != DuplicateByNormalizedURL {
		t.Fatalf("unexpected reason %q", matches[0].Reason)
	}
}

func TestDeduplicateArticlesByContentFingerprint(t *testing.T) {
	published := time.Date(2026, 9, 19, 12, 30, 0, 0, time.UTC)
	articles := []Article{
		{
			FeedID:      "feed-1",
			Identifier:  "old-id",
			URL:         "https://example.test/old",
			Title:       "Same Story",
			Author:      "Ada Example",
			PublishedAt: &published,
			Content:     "<p>Same content</p>",
		},
		{
			FeedID:      "feed-1",
			Identifier:  "new-id",
			URL:         "https://example.test/new",
			Title:       " same   story ",
			Author:      "ADA EXAMPLE",
			PublishedAt: &published,
			Content:     "<p>Same content</p>",
		},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 1 || len(matches) != 1 {
		t.Fatalf("expected content duplicate: unique=%d matches=%v", len(unique), matches)
	}
	if matches[0].Reason != DuplicateByContentFingerprint {
		t.Fatalf("unexpected reason %q", matches[0].Reason)
	}
}

func TestDeduplicateArticlesRegistersAliasesFromDuplicate(t *testing.T) {
	articles := []Article{
		{FeedID: "feed-1", Identifier: "old-id", URL: "https://example.test/article"},
		{FeedID: "feed-1", Identifier: "new-id", URL: "https://example.test/article"},
		{FeedID: "feed-1", Identifier: "new-id", URL: "https://example.test/article-v2"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 1 || len(matches) != 2 {
		t.Fatalf("expected alias propagation: unique=%d matches=%v", len(unique), matches)
	}
	if matches[1].CanonicalIndex != 0 || matches[1].Reason != DuplicateBySourceIdentifier {
		t.Fatalf("unexpected alias match: %+v", matches[1])
	}
}

func TestDeduplicateArticlesRetainsConflictingEvidence(t *testing.T) {
	articles := []Article{
		{FeedID: "feed-1", Identifier: "id-a", URL: "https://example.test/a"},
		{FeedID: "feed-1", Identifier: "id-b", URL: "https://example.test/b"},
		{FeedID: "feed-1", Identifier: "id-a", URL: "https://example.test/b"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 3 {
		t.Fatalf("conflicting evidence must remain distinct, got %d unique", len(unique))
	}
	if len(matches) != 0 {
		t.Fatalf("conflicting evidence must not produce a duplicate match: %v", matches)
	}
}

func TestDeduplicateArticlesRequiresEnoughFingerprintEvidence(t *testing.T) {
	articles := []Article{
		{FeedID: "feed-1", Title: "Untimed story", Content: "same"},
		{FeedID: "feed-1", Title: "Untimed story", Content: "same"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 2 || len(matches) != 0 {
		t.Fatalf("articles without publication time must remain distinct: unique=%d matches=%v", len(unique), matches)
	}
}

func TestDeduplicateArticlesRequiresSourceScope(t *testing.T) {
	published := time.Date(2026, 9, 19, 12, 30, 0, 0, time.UTC)
	articles := []Article{
		{Identifier: "same", URL: "https://example.test/post", Title: "Same", PublishedAt: &published, Content: "same"},
		{Identifier: "same", URL: "https://example.test/post", Title: "Same", PublishedAt: &published, Content: "same"},
	}

	unique, matches := DeduplicateArticles(articles)
	if len(unique) != 2 || len(matches) != 0 {
		t.Fatalf("unscoped articles must remain distinct: unique=%d matches=%v", len(unique), matches)
	}
}
