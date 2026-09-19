package feed

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseRSSNormalizesCoreFields(t *testing.T) {
	input := []byte(`<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Example Feed</title>
    <description>Feed description</description>
    <link>https://example.test/</link>
    <language>en-US</language>
    <image><url>https://example.test/feed.png</url></image>
    <item>
      <guid>article-1</guid>
      <title>Article One</title>
      <link>https://example.test/articles/1</link>
      <author>Ada Example</author>
      <pubDate>Fri, 18 Sep 2026 12:30:00 +0000</pubDate>
      <description>Summary text</description>
      <content:encoded><![CDATA[<p>Full content</p>]]></content:encoded>
      <category>News</category>
      <category>GoreeCloud</category>
      <media:content url="https://example.test/audio.mp3" type="audio/mpeg" />
      <media:thumbnail url="https://example.test/thumb.jpg" width="320" height="180" />
    </item>
  </channel>
</rss>`)

	got, err := Parse(input, "https://example.test/feed.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Feed.Title != "Example Feed" || got.Feed.Description != "Feed description" {
		t.Fatalf("unexpected feed: %+v", got.Feed)
	}
	if got.Feed.ImageURL != "https://example.test/feed.png" || got.Feed.Language != "en-US" {
		t.Fatalf("unexpected feed metadata: %+v", got.Feed)
	}
	if got.Feed.Source.Format != "rss" || got.Feed.Source.URL != "https://example.test/feed.xml" {
		t.Fatalf("unexpected source info: %+v", got.Feed.Source)
	}
	if len(got.Articles) != 1 {
		t.Fatalf("expected one article, got %d", len(got.Articles))
	}
	a := got.Articles[0]
	if a.Identifier != "article-1" || a.URL != "https://example.test/articles/1" || a.Author != "Ada Example" {
		t.Fatalf("unexpected article identity: %+v", a)
	}
	if a.Content != "<p>Full content</p>" || a.Summary != "Summary text" {
		t.Fatalf("unexpected article content: %+v", a)
	}
	if a.PublishedAt == nil || !a.PublishedAt.Equal(time.Date(2026, 9, 18, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected publication time: %v", a.PublishedAt)
	}
	if len(a.Categories) != 2 || a.Categories[0] != "News" || a.Categories[1] != "GoreeCloud" {
		t.Fatalf("unexpected categories: %v", a.Categories)
	}
	if len(a.Media) != 1 || a.Media[0].URL != "https://example.test/audio.mp3" {
		t.Fatalf("unexpected media: %+v", a.Media)
	}
	if len(a.Images) != 1 || a.Images[0].URL != "https://example.test/thumb.jpg" || a.Images[0].Width != 320 {
		t.Fatalf("unexpected images: %+v", a.Images)
	}
}

func TestParseAtomNormalizesCoreFields(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xml:lang="en">
  <title>Atom Example</title>
  <subtitle>Atom description</subtitle>
  <link href="https://example.test/" rel="alternate" />
  <icon>https://example.test/icon.png</icon>
  <logo>https://example.test/logo.png</logo>
  <entry>
    <id>urn:example:2</id>
    <title>Atom Article</title>
    <link href="https://example.test/articles/2" rel="alternate" />
    <link href="https://example.test/image.webp" rel="enclosure" type="image/webp" />
    <author><name>Grace Example</name></author>
    <published>2026-09-18T13:00:00Z</published>
    <updated>2026-09-18T14:00:00Z</updated>
    <summary>Atom summary</summary>
    <content type="html">&lt;p&gt;Atom content&lt;/p&gt;</content>
    <category term="Technology" />
  </entry>
</feed>`)

	got, err := Parse(input, "https://example.test/atom.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Feed.Title != "Atom Example" || got.Feed.IconURL != "https://example.test/icon.png" ||
		got.Feed.ImageURL != "https://example.test/logo.png" || got.Feed.Language != "en" {
		t.Fatalf("unexpected feed: %+v", got.Feed)
	}
	if len(got.Articles) != 1 {
		t.Fatalf("expected one article, got %d", len(got.Articles))
	}
	a := got.Articles[0]
	if a.Identifier != "urn:example:2" || a.Author != "Grace Example" || a.Content != "<p>Atom content</p>" {
		t.Fatalf("unexpected article: %+v", a)
	}
	if a.UpdatedAt == nil || a.UpdatedAt.Format(time.RFC3339) != "2026-09-18T14:00:00Z" {
		t.Fatalf("unexpected updated time: %v", a.UpdatedAt)
	}
	if len(a.Images) != 1 || a.Images[0].URL != "https://example.test/image.webp" {
		t.Fatalf("unexpected enclosure image: %+v", a.Images)
	}
}

func TestParseRecoversIncompleteArticle(t *testing.T) {
	input := []byte(`<rss><channel><title>Partial</title><item><title>Usable title</title></item></channel></rss>`)
	got, err := Parse(input, "https://example.test/partial.xml")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(got.Articles) != 1 || got.Articles[0].Title != "Usable title" {
		t.Fatalf("unexpected partial result: %+v", got)
	}
}

func TestParseWarnsInsteadOfFailingForBadDate(t *testing.T) {
	input := []byte(`<rss><channel><title>Dates</title><item><title>A</title><pubDate>not-a-date</pubDate></item></channel></rss>`)
	got, err := Parse(input, "")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Articles[0].PublishedAt != nil {
		t.Fatalf("unexpected parsed date: %v", got.Articles[0].PublishedAt)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected a recoverable date warning")
	}
}

func TestParseRejectsUnsupportedAndUnsafeXML(t *testing.T) {
	t.Run("unsupported format", func(t *testing.T) {
		_, err := Parse([]byte(`<opml version="2.0"></opml>`), "")
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
		}
	})

	t.Run("doctype", func(t *testing.T) {
		_, err := Parse([]byte(`<!DOCTYPE rss [<!ENTITY x "unsafe">]><rss><channel><title>&x;</title></channel></rss>`), "")
		if !errors.Is(err, ErrUnsafeXMLDirective) {
			t.Fatalf("expected ErrUnsafeXMLDirective, got %v", err)
		}
	})
}

func TestArticleStateRemainsSeparateFromArticleContent(t *testing.T) {
	article := Article{Identifier: "shared-entry", Title: "Shared content"}
	alice := ArticleState{UserID: "alice", ArticleID: "article-1", Read: true}
	bob := ArticleState{UserID: "bob", ArticleID: "article-1", Saved: true}

	if article.Title != "Shared content" || !alice.Read || bob.Read || !bob.Saved {
		t.Fatalf("article/state separation failed: article=%+v alice=%+v bob=%+v", article, alice, bob)
	}
}

func TestComplexityLimit(t *testing.T) {
	var b strings.Builder
	for range maxXMLDepth + 1 {
		b.WriteString("<x>")
	}
	for range maxXMLDepth + 1 {
		b.WriteString("</x>")
	}
	_, err := Parse([]byte(b.String()), "")
	if !errors.Is(err, ErrXMLTooComplex) {
		t.Fatalf("expected ErrXMLTooComplex, got %v", err)
	}
}
