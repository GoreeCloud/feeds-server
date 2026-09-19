package feed

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	maxXMLDepth = 64
	maxXMLNodes = 100000
)

var (
	ErrEmptyDocument      = errors.New("feed document is empty")
	ErrUnsupportedFormat  = errors.New("unsupported feed format")
	ErrUnsafeXMLDirective = errors.New("unsafe XML directive")
	ErrXMLTooComplex      = errors.New("XML document exceeds parser complexity limits")
)

type xmlNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     strings.Builder
	Children []*xmlNode
}

func (n *xmlNode) child(local string) *xmlNode {
	for _, child := range n.Children {
		if strings.EqualFold(child.Name.Local, local) {
			return child
		}
	}
	return nil
}

func (n *xmlNode) children(local string) []*xmlNode {
	var out []*xmlNode
	for _, child := range n.Children {
		if strings.EqualFold(child.Name.Local, local) {
			out = append(out, child)
		}
	}
	return out
}

func (n *xmlNode) value() string {
	return strings.TrimSpace(n.Text.String())
}

func (n *xmlNode) attr(local string) string {
	for _, attr := range n.Attrs {
		if strings.EqualFold(attr.Name.Local, local) {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

// Parse normalizes UTF-8 RSS 2.x or Atom 1.x XML into GoreeCloud Feeds
// internal models. Network retrieval, persistence, HTML sanitization, and
// authentication are intentionally outside this parser boundary.
func Parse(data []byte, sourceURL string) (ParsedFeed, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return ParsedFeed{}, ErrEmptyDocument
	}

	root, warnings, err := decodeTree(data)
	if err != nil {
		return ParsedFeed{}, err
	}

	switch strings.ToLower(root.Name.Local) {
	case "rss":
		result, err := parseRSS(root, sourceURL)
		result.Warnings = append(warnings, result.Warnings...)
		return result, err
	case "feed":
		result, err := parseAtom(root, sourceURL)
		result.Warnings = append(warnings, result.Warnings...)
		return result, err
	default:
		return ParsedFeed{}, fmt.Errorf("%w: root element %q", ErrUnsupportedFormat, root.Name.Local)
	}
}

func decodeTree(data []byte) (*xmlNode, []string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	var root *xmlNode
	var stack []*xmlNode
	nodes := 0
	var warnings []string

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, warnings, fmt.Errorf("parse XML: %w", err)
		}

		switch t := token.(type) {
		case xml.Directive:
			if strings.Contains(strings.ToUpper(string(t)), "DOCTYPE") ||
				strings.Contains(strings.ToUpper(string(t)), "ENTITY") {
				return nil, warnings, ErrUnsafeXMLDirective
			}
		case xml.StartElement:
			if len(stack)+1 > maxXMLDepth {
				return nil, warnings, ErrXMLTooComplex
			}
			nodes++
			if nodes > maxXMLNodes {
				return nil, warnings, ErrXMLTooComplex
			}
			node := &xmlNode{Name: t.Name, Attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) == 0 {
				if root != nil {
					return nil, warnings, errors.New("multiple XML root elements")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				_, _ = stack[len(stack)-1].Text.Write([]byte(t))
			}
		}
	}

	if root == nil {
		return nil, warnings, ErrEmptyDocument
	}
	if len(stack) != 0 {
		warnings = append(warnings, "feed XML ended before all elements were explicitly closed")
	}
	return root, warnings, nil
}

func parseRSS(root *xmlNode, sourceURL string) (ParsedFeed, error) {
	channel := root.child("channel")
	if channel == nil {
		return ParsedFeed{}, errors.New("RSS feed has no channel element")
	}

	result := ParsedFeed{
		Feed: Feed{
			URL:         firstNonEmpty(childValue(channel, "link"), sourceURL),
			Title:       childValue(channel, "title"),
			Description: childValue(channel, "description"),
			Language:    childValue(channel, "language"),
			Source:      SourceInfo{Format: "rss", URL: sourceURL},
		},
	}

	if image := channel.child("image"); image != nil {
		result.Feed.ImageURL = childValue(image, "url")
	}
	if result.Feed.ImageURL == "" {
		for _, node := range channel.children("image") {
			if href := node.attr("href"); href != "" {
				result.Feed.ImageURL = href
				break
			}
		}
	}

	for _, item := range channel.children("item") {
		article := Article{
			Identifier: childValue(item, "guid"),
			URL:        childValue(item, "link"),
			Title:      childValue(item, "title"),
			Author:     firstNonEmpty(childValue(item, "author"), childValue(item, "creator")),
			Summary:    childValue(item, "description"),
			Content:    childValue(item, "encoded"),
			Language:   firstNonEmpty(childValue(item, "language"), result.Feed.Language),
			Source:     SourceInfo{Format: "rss", URL: sourceURL},
		}
		if article.Identifier == "" {
			article.Identifier = article.URL
		}
		if article.Content == "" {
			article.Content = article.Summary
		}

		if raw := firstNonEmpty(childValue(item, "pubDate"), childValue(item, "published")); raw != "" {
			if parsed, ok := parseDate(raw); ok {
				article.PublishedAt = &parsed
			} else {
				result.Warnings = append(result.Warnings, "article publication date could not be normalized")
			}
		}
		if raw := firstNonEmpty(childValue(item, "updated"), childValue(item, "modified")); raw != "" {
			if parsed, ok := parseDate(raw); ok {
				article.UpdatedAt = &parsed
			} else {
				result.Warnings = append(result.Warnings, "article updated date could not be normalized")
			}
		}

		for _, category := range item.children("category") {
			addUnique(&article.Categories, category.value())
		}
		for _, tag := range item.children("tag") {
			addUnique(&article.Tags, tag.value())
		}
		for _, keywords := range item.children("keywords") {
			for _, keyword := range strings.Split(keywords.value(), ",") {
				addUnique(&article.Tags, keyword)
			}
		}

		for _, enclosure := range item.children("enclosure") {
			addMediaOrImage(&article, enclosure.attr("url"), enclosure.attr("type"), 0, 0, "")
		}
		for _, media := range item.children("content") {
			if media.attr("url") == "" {
				continue
			}
			addMediaOrImage(
				&article,
				media.attr("url"),
				media.attr("type"),
				parsePositiveInt(media.attr("width")),
				parsePositiveInt(media.attr("height")),
				media.attr("medium"),
			)
		}
		for _, thumbnail := range item.children("thumbnail") {
			addImage(&article.Images, Image{
				URL:    thumbnail.attr("url"),
				Width:  parsePositiveInt(thumbnail.attr("width")),
				Height: parsePositiveInt(thumbnail.attr("height")),
			})
		}

		if article.Title == "" && article.URL == "" && article.Identifier == "" && article.Content == "" {
			result.Warnings = append(result.Warnings, "empty RSS item skipped")
			continue
		}
		result.Articles = append(result.Articles, article)
	}

	if result.Feed.Title == "" {
		result.Warnings = append(result.Warnings, "feed title is missing")
	}
	return result, nil
}

func parseAtom(root *xmlNode, sourceURL string) (ParsedFeed, error) {
	result := ParsedFeed{
		Feed: Feed{
			URL:         atomLink(root, "alternate"),
			Title:       childValue(root, "title"),
			Description: firstNonEmpty(childValue(root, "subtitle"), childValue(root, "summary")),
			IconURL:     childValue(root, "icon"),
			ImageURL:    childValue(root, "logo"),
			Language:    firstNonEmpty(root.attr("lang"), root.attr("language")),
			Source:      SourceInfo{Format: "atom", URL: sourceURL},
		},
	}
	if result.Feed.URL == "" {
		result.Feed.URL = sourceURL
	}

	for _, entry := range root.children("entry") {
		article := Article{
			Identifier: childValue(entry, "id"),
			URL:        atomLink(entry, "alternate"),
			Title:      childValue(entry, "title"),
			Author:     atomAuthor(entry),
			Summary:    childValue(entry, "summary"),
			Content:    childValue(entry, "content"),
			Language:   firstNonEmpty(entry.attr("lang"), result.Feed.Language),
			Source:     SourceInfo{Format: "atom", URL: sourceURL},
		}
		if article.Identifier == "" {
			article.Identifier = article.URL
		}
		if article.Content == "" {
			article.Content = article.Summary
		}

		if raw := childValue(entry, "published"); raw != "" {
			if parsed, ok := parseDate(raw); ok {
				article.PublishedAt = &parsed
			} else {
				result.Warnings = append(result.Warnings, "article publication date could not be normalized")
			}
		}
		if raw := childValue(entry, "updated"); raw != "" {
			if parsed, ok := parseDate(raw); ok {
				article.UpdatedAt = &parsed
			} else {
				result.Warnings = append(result.Warnings, "article updated date could not be normalized")
			}
		}

		for _, category := range entry.children("category") {
			term := firstNonEmpty(category.attr("term"), category.value())
			addUnique(&article.Categories, term)
		}
		for _, link := range entry.children("link") {
			rel := strings.ToLower(firstNonEmpty(link.attr("rel"), "alternate"))
			if rel != "enclosure" {
				continue
			}
			addMediaOrImage(
				&article,
				link.attr("href"),
				link.attr("type"),
				0,
				0,
				"",
			)
		}

		if article.Title == "" && article.URL == "" && article.Identifier == "" && article.Content == "" {
			result.Warnings = append(result.Warnings, "empty Atom entry skipped")
			continue
		}
		result.Articles = append(result.Articles, article)
	}

	if result.Feed.Title == "" {
		result.Warnings = append(result.Warnings, "feed title is missing")
	}
	return result, nil
}

func childValue(node *xmlNode, local string) string {
	if node == nil {
		return ""
	}
	child := node.child(local)
	if child == nil {
		return ""
	}
	return child.value()
}

func atomLink(node *xmlNode, wantedRel string) string {
	var fallback string
	for _, link := range node.children("link") {
		href := link.attr("href")
		if href == "" {
			continue
		}
		rel := strings.ToLower(firstNonEmpty(link.attr("rel"), "alternate"))
		if rel == wantedRel {
			return href
		}
		if fallback == "" {
			fallback = href
		}
	}
	return fallback
}

func atomAuthor(node *xmlNode) string {
	author := node.child("author")
	if author == nil {
		return ""
	}
	return firstNonEmpty(childValue(author, "name"), childValue(author, "email"), author.value())
}

func parseDate(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func addMediaOrImage(article *Article, url, mimeType string, width, height int, medium string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	if strings.EqualFold(medium, "image") || strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		addImage(&article.Images, Image{URL: url, Width: width, Height: height})
		return
	}
	addMedia(&article.Media, Media{
		URL:      url,
		MIMEType: strings.TrimSpace(mimeType),
		Width:    width,
		Height:   height,
	})
}

func addMedia(target *[]Media, value Media) {
	for _, existing := range *target {
		if existing.URL == value.URL {
			return
		}
	}
	*target = append(*target, value)
}

func addImage(target *[]Image, value Image) {
	value.URL = strings.TrimSpace(value.URL)
	if value.URL == "" {
		return
	}
	for _, existing := range *target {
		if existing.URL == value.URL {
			return
		}
	}
	*target = append(*target, value)
}

func addUnique(target *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	for _, existing := range *target {
		if strings.EqualFold(existing, value) {
			return
		}
	}
	*target = append(*target, value)
}

func parsePositiveInt(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
