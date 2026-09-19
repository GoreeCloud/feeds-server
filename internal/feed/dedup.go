package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"
	"strings"
)

type DuplicateReason string

const (
	DuplicateBySourceIdentifier  DuplicateReason = "source_identifier"
	DuplicateByNormalizedURL     DuplicateReason = "normalized_url"
	DuplicateByContentFingerprint DuplicateReason = "content_fingerprint"
)

type DuplicateMatch struct {
	DuplicateIndex int
	CanonicalIndex int
	Reason         DuplicateReason
}

type articleKeys struct {
	identifier  string
	url         string
	fingerprint string
}

// DeduplicateArticles conservatively collapses duplicate articles while
// preserving the first encountered article as canonical.
//
// The initial Development tranche requires a source scope (FeedID or Source.URL)
// and uses only exact, explainable evidence:
//   - source-scoped identifiers;
//   - source-scoped conservatively normalized URLs; and
//   - source-scoped title/publication/content fingerprints.
//
// If independent keys point at different canonical articles, the candidate is
// retained instead of being merged. Fuzzy similarity is intentionally deferred.
func DeduplicateArticles(articles []Article) ([]Article, []DuplicateMatch) {
	unique := make([]Article, 0, len(articles))
	matches := make([]DuplicateMatch, 0)

	identifierIndex := make(map[string]int)
	urlIndex := make(map[string]int)
	fingerprintIndex := make(map[string]int)

	for originalIndex, article := range articles {
		keys := keysForArticle(article)

		canonical, reason, ok := resolveCanonical(keys, identifierIndex, urlIndex, fingerprintIndex)
		if ok {
			matches = append(matches, DuplicateMatch{
				DuplicateIndex: originalIndex,
				CanonicalIndex: canonical,
				Reason:         reason,
			})
			registerKeys(keys, canonical, identifierIndex, urlIndex, fingerprintIndex)
			continue
		}

		canonical = len(unique)
		unique = append(unique, article)
		registerKeys(keys, canonical, identifierIndex, urlIndex, fingerprintIndex)
	}

	return unique, matches
}

func keysForArticle(article Article) articleKeys {
	scope := sourceScope(article)
	if scope == "" {
		return articleKeys{}
	}

	keys := articleKeys{}

	if id := normalizeToken(article.Identifier); id != "" {
		keys.identifier = scope + "|id|" + id
	}

	if normalized := normalizeArticleURL(article.URL); normalized != "" {
		keys.url = scope + "|url|" + normalized
	}

	if fingerprint := articleContentFingerprint(article); fingerprint != "" {
		keys.fingerprint = scope + "|content|" + fingerprint
	}

	return keys
}

func resolveCanonical(
	keys articleKeys,
	identifierIndex map[string]int,
	urlIndex map[string]int,
	fingerprintIndex map[string]int,
) (int, DuplicateReason, bool) {
	type candidate struct {
		index  int
		reason DuplicateReason
	}

	var candidates []candidate
	if keys.identifier != "" {
		if index, ok := identifierIndex[keys.identifier]; ok {
			candidates = append(candidates, candidate{index: index, reason: DuplicateBySourceIdentifier})
		}
	}
	if keys.url != "" {
		if index, ok := urlIndex[keys.url]; ok {
			candidates = append(candidates, candidate{index: index, reason: DuplicateByNormalizedURL})
		}
	}
	if keys.fingerprint != "" {
		if index, ok := fingerprintIndex[keys.fingerprint]; ok {
			candidates = append(candidates, candidate{index: index, reason: DuplicateByContentFingerprint})
		}
	}
	if len(candidates) == 0 {
		return 0, "", false
	}

	canonical := candidates[0].index
	for _, candidate := range candidates[1:] {
		if candidate.index != canonical {
			return 0, "", false
		}
	}

	// Candidates are appended strongest-first.
	return canonical, candidates[0].reason, true
}

func registerKeys(
	keys articleKeys,
	canonical int,
	identifierIndex map[string]int,
	urlIndex map[string]int,
	fingerprintIndex map[string]int,
) {
	registerKey(identifierIndex, keys.identifier, canonical)
	registerKey(urlIndex, keys.url, canonical)
	registerKey(fingerprintIndex, keys.fingerprint, canonical)
}

func registerKey(index map[string]int, key string, canonical int) {
	if key == "" {
		return
	}
	if _, exists := index[key]; exists {
		return
	}
	index[key] = canonical
}

func sourceScope(article Article) string {
	if article.FeedID != "" {
		return "feed:" + string(article.FeedID)
	}
	if normalized := normalizeArticleURL(article.Source.URL); normalized != "" {
		return "source:" + normalized
	}
	return ""
}

func normalizeArticleURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return ""
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}

	if strings.Contains(hostname, ":") {
		hostname = "[" + hostname + "]"
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(strings.Trim(hostname, "[]"), port)
	} else {
		parsed.Host = hostname
	}

	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.RawQuery = parsed.Query().Encode()

	return parsed.String()
}

func articleContentFingerprint(article Article) string {
	title := normalizeToken(article.Title)
	if title == "" || article.PublishedAt == nil {
		return ""
	}

	content := article.Content
	if strings.TrimSpace(content) == "" {
		content = article.Summary
	}
	content = normalizeContent(content)
	if content == "" {
		return ""
	}

	author := normalizeToken(article.Author)
	published := article.PublishedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")

	sum := sha256.Sum256([]byte(strings.Join([]string{
		title,
		author,
		published,
		content,
	}, "\n")))
	return hex.EncodeToString(sum[:])
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func normalizeContent(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
