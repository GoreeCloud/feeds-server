package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/GoreeCloud/feeds-server/internal/feed"
)

func TestRepositoryPersistsCoreFeedArticleAndUserState(t *testing.T) {
	store := repositoryTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := UserReference{ID: "repo-user", IdentitySubject: "identity:repo-user"}
	if err := store.UpsertUserReference(ctx, user); err != nil {
		t.Fatalf("UpsertUserReference() error = %v", err)
	}

	seenAt := time.Date(2026, 9, 19, 19, 45, 0, 0, time.UTC)
	feedWrite := FeedWrite{
		Feed: feed.Feed{
			ID:          "repo-feed",
			URL:         "https://example.test/feed.xml",
			Title:       "Example Feed",
			Description: "Repository integration feed",
			Language:    "en",
			Source:      feed.SourceInfo{Format: "rss", URL: "https://example.test/feed.xml"},
		},
		NormalizedURL: "https://example.test/feed.xml",
		SeenAt:        seenAt,
	}
	if err := store.UpsertFeed(ctx, feedWrite); err != nil {
		t.Fatalf("UpsertFeed() error = %v", err)
	}

	subscription := SubscriptionWrite{
		Subscription: feed.Subscription{
			ID:        "repo-subscription",
			UserID:    user.ID,
			FeedID:    feedWrite.Feed.ID,
			CreatedAt: seenAt,
		},
	}
	if err := store.UpsertSubscription(ctx, subscription); err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}

	publishedAt := seenAt.Add(-2 * time.Hour)
	sourceUpdatedAt := seenAt.Add(-time.Hour)
	firstRetrieved := seenAt
	firstArticle := ArticleWrite{
		Article: feed.Article{
			ID:          "repo-article",
			FeedID:      feedWrite.Feed.ID,
			Identifier:  "entry-1",
			URL:         "https://example.test/articles/1",
			Title:       "First Title",
			Author:      "Ada Example",
			PublishedAt: &publishedAt,
			UpdatedAt:   &sourceUpdatedAt,
			Summary:     "summary",
			Content:     "content-v1",
			Language:    "en",
			Source:      feed.SourceInfo{Format: "rss", URL: feedWrite.Feed.URL},
		},
		NormalizedURL:      "https://example.test/articles/1",
		ContentFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RetrievedAt:        firstRetrieved,
		SourceHistoryID:    "repo-history-1",
		IdentityKeys: []ArticleIdentityKey{
			{Type: feed.DuplicateBySourceIdentifier, Value: "entry-1"},
			{Type: feed.DuplicateByNormalizedURL, Value: "https://example.test/articles/1"},
		},
	}
	if err := store.UpsertArticle(ctx, firstArticle); err != nil {
		t.Fatalf("first UpsertArticle() error = %v", err)
	}

	secondRetrieved := seenAt.Add(10 * time.Minute)
	secondArticle := firstArticle
	secondArticle.Article.Title = "Updated Title"
	secondArticle.Article.Content = "content-v2"
	secondArticle.ContentFingerprint = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	secondArticle.RetrievedAt = secondRetrieved
	secondArticle.SourceHistoryID = "repo-history-2"
	secondArticle.IdentityKeys = append(
		append([]ArticleIdentityKey{}, firstArticle.IdentityKeys...),
		ArticleIdentityKey{Type: feed.DuplicateByContentFingerprint, Value: secondArticle.ContentFingerprint},
	)
	if err := store.UpsertArticle(ctx, secondArticle); err != nil {
		t.Fatalf("second UpsertArticle() error = %v", err)
	}

	stored, err := store.ArticleByID(ctx, firstArticle.Article.ID)
	if err != nil {
		t.Fatalf("ArticleByID() error = %v", err)
	}
	if stored.Article.Title != "Updated Title" || stored.Article.Content != "content-v2" {
		t.Fatalf("article was not updated: %+v", stored.Article)
	}
	if !stored.FirstRetrievedAt.Equal(firstRetrieved) {
		t.Fatalf("first retrieval changed: got %v want %v", stored.FirstRetrievedAt, firstRetrieved)
	}
	if !stored.LastRetrievedAt.Equal(secondRetrieved) {
		t.Fatalf("last retrieval mismatch: got %v want %v", stored.LastRetrievedAt, secondRetrieved)
	}
	if stored.ContentFingerprint != secondArticle.ContentFingerprint {
		t.Fatalf("content fingerprint mismatch: %q", stored.ContentFingerprint)
	}

	lastReadAt := secondRetrieved.Add(time.Minute)
	stateWrite := ArticleStateWrite{
		State: feed.ArticleState{
			UserID:       user.ID,
			ArticleID:    firstArticle.Article.ID,
			Read:         true,
			Saved:        true,
			Favorite:     true,
			ReadPosition: 0.5,
			UpdatedAt:    lastReadAt,
		},
		Preserved:  true,
		LastReadAt: &lastReadAt,
	}
	if err := store.UpsertArticleState(ctx, stateWrite); err != nil {
		t.Fatalf("UpsertArticleState() error = %v", err)
	}

	state, err := store.ArticleStateForUser(ctx, user.ID, firstArticle.Article.ID)
	if err != nil {
		t.Fatalf("ArticleStateForUser() error = %v", err)
	}
	if !state.State.Read || !state.State.Saved || !state.State.Favorite || !state.Preserved {
		t.Fatalf("unexpected article state: %+v", state)
	}
	if state.State.ReadPosition != 0.5 {
		t.Fatalf("unexpected read position %v", state.State.ReadPosition)
	}

	var historyCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM goreecloud_feeds.article_source_history
		WHERE article_id = $1
	`, firstArticle.Article.ID).Scan(&historyCount); err != nil {
		t.Fatalf("count source history: %v", err)
	}
	if historyCount != 2 {
		t.Fatalf("expected 2 source-history rows, got %d", historyCount)
	}

	var keyCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM goreecloud_feeds.article_identity_keys
		WHERE article_id = $1
	`, firstArticle.Article.ID).Scan(&keyCount); err != nil {
		t.Fatalf("count identity keys: %v", err)
	}
	if keyCount != 3 {
		t.Fatalf("expected 3 identity keys, got %d", keyCount)
	}
}

func TestRepositoryRejectsIdentityAndDeduplicationRebinding(t *testing.T) {
	store := repositoryTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.UpsertUserReference(ctx, UserReference{ID: "guard-user", IdentitySubject: "identity:guard-a"}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.UpsertUserReference(ctx, UserReference{ID: "guard-user", IdentitySubject: "identity:guard-b"}); err == nil {
		t.Fatal("expected identity subject rebinding to fail")
	}

	now := time.Date(2026, 9, 19, 20, 0, 0, 0, time.UTC)
	for _, id := range []feed.ID{"guard-feed-a", "guard-feed-b"} {
		if err := store.UpsertFeed(ctx, FeedWrite{
			Feed: feed.Feed{ID: id, URL: "https://example.test/" + string(id), Source: feed.SourceInfo{Format: "rss"}},
			NormalizedURL: "https://example.test/" + string(id),
			SeenAt:        now,
		}); err != nil {
			t.Fatalf("create feed %s: %v", id, err)
		}
	}

	base := ArticleWrite{
		Article: feed.Article{
			ID:         "guard-article-a",
			FeedID:     "guard-feed-a",
			Identifier: "shared-key",
			URL:        "https://example.test/a",
			Title:      "A",
		},
		NormalizedURL:   "https://example.test/a",
		RetrievedAt:     now,
		SourceHistoryID: "guard-history-a",
		IdentityKeys: []ArticleIdentityKey{
			{Type: feed.DuplicateBySourceIdentifier, Value: "shared-key"},
		},
	}
	if err := store.UpsertArticle(ctx, base); err != nil {
		t.Fatalf("create base article: %v", err)
	}

	reboundID := base
	reboundID.Article.FeedID = "guard-feed-b"
	reboundID.SourceHistoryID = "guard-history-rebound"
	if err := store.UpsertArticle(ctx, reboundID); err == nil {
		t.Fatal("expected article id feed rebinding to fail")
	}

	second := base
	second.Article.ID = "guard-article-b"
	second.Article.URL = "https://example.test/b"
	second.NormalizedURL = "https://example.test/b"
	second.SourceHistoryID = "guard-history-b"
	if err := store.UpsertArticle(ctx, second); err == nil {
		t.Fatal("expected identity-key rebinding to another article to fail")
	}

	var secondCount int
	if err := store.pool.QueryRow(
		ctx,
		"SELECT count(*) FROM goreecloud_feeds.articles WHERE id = $1",
		second.Article.ID,
	).Scan(&secondCount); err != nil {
		t.Fatalf("count rolled-back article: %v", err)
	}
	if secondCount != 0 {
		t.Fatalf("conflicting identity-key transaction was not rolled back; count=%d", secondCount)
	}
}

func TestRepositoryRejectsUnsupportedArticleCollectionsAndBadReadPosition(t *testing.T) {
	store := repositoryTestStore(t)
	ctx := context.Background()

	err := store.UpsertArticle(ctx, ArticleWrite{
		Article: feed.Article{
			ID:         "unsupported-article",
			FeedID:     "unsupported-feed",
			Categories: []string{"news"},
		},
		RetrievedAt:     time.Now().UTC(),
		SourceHistoryID: "unsupported-history",
	})
	if !errors.Is(err, ErrUnsupportedArticleMetadata) {
		t.Fatalf("expected ErrUnsupportedArticleMetadata, got %v", err)
	}

	err = store.UpsertArticleState(ctx, ArticleStateWrite{
		State: feed.ArticleState{
			UserID:       "user",
			ArticleID:    "article",
			ReadPosition: 1.5,
			UpdatedAt:    time.Now().UTC(),
		},
	})
	if err == nil {
		t.Fatal("expected invalid read position to fail")
	}
}

func repositoryTestStore(t *testing.T) *Store {
	t.Helper()
	connectionString := os.Getenv("FEEDS_TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("FEEDS_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	store, err := Open(ctx, connectionString)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(store.Close)

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return store
}
