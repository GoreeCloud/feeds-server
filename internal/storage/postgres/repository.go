package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoreeCloud/feeds-server/internal/feed"
	"github.com/jackc/pgx/v5"
)

var ErrUnsupportedArticleMetadata = errors.New("article categories, tags, media, and images are not yet supported by the PostgreSQL schema")

type UserReference struct {
	ID              feed.ID
	IdentitySubject string
}

type FeedWrite struct {
	Feed          feed.Feed
	NormalizedURL string
	SeenAt        time.Time
}

type SubscriptionWrite struct {
	Subscription feed.Subscription
	Disabled     bool
}

type ArticleIdentityKey struct {
	Type  feed.DuplicateReason
	Value string
}

type ArticleWrite struct {
	Article             feed.Article
	NormalizedURL       string
	ContentFingerprint  string
	RetrievedAt         time.Time
	RetentionEligibleAt *time.Time
	SourceHistoryID     feed.ID
	IdentityKeys        []ArticleIdentityKey
}

type StoredArticle struct {
	Article             feed.Article
	NormalizedURL       string
	ContentFingerprint  string
	FirstRetrievedAt    time.Time
	LastRetrievedAt     time.Time
	RetentionEligibleAt *time.Time
}

type ArticleStateWrite struct {
	State      feed.ArticleState
	Preserved  bool
	LastReadAt *time.Time
}

type StoredArticleState struct {
	State      feed.ArticleState
	Preserved  bool
	LastReadAt *time.Time
}

func (s *Store) UpsertUserReference(ctx context.Context, user UserReference) error {
	if err := ensureStore(s); err != nil {
		return err
	}
	if strings.TrimSpace(string(user.ID)) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(user.IdentitySubject) == "" {
		return fmt.Errorf("identity subject is required")
	}

	var storedSubject string
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO goreecloud_feeds.users (id, identity_subject)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET updated_at = CURRENT_TIMESTAMP
		RETURNING identity_subject
	`, user.ID, user.IdentitySubject).Scan(&storedSubject); err != nil {
		return fmt.Errorf("upsert user reference: %w", err)
	}
	if storedSubject != user.IdentitySubject {
		return fmt.Errorf("user id %q is already bound to a different identity subject", user.ID)
	}
	return nil
}

func (s *Store) UpsertFeed(ctx context.Context, input FeedWrite) error {
	if err := ensureStore(s); err != nil {
		return err
	}
	if strings.TrimSpace(string(input.Feed.ID)) == "" {
		return fmt.Errorf("feed id is required")
	}
	if strings.TrimSpace(input.Feed.URL) == "" {
		return fmt.Errorf("feed canonical url is required")
	}
	if strings.TrimSpace(input.NormalizedURL) == "" {
		return fmt.Errorf("feed normalized url is required")
	}
	if input.SeenAt.IsZero() {
		return fmt.Errorf("feed seen time is required")
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO goreecloud_feeds.feeds (
			id, canonical_url, normalized_url, title, description, icon_url,
			image_url, language, source_format, source_url, first_seen_at,
			last_seen_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			canonical_url = EXCLUDED.canonical_url,
			normalized_url = EXCLUDED.normalized_url,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			icon_url = EXCLUDED.icon_url,
			image_url = EXCLUDED.image_url,
			language = EXCLUDED.language,
			source_format = EXCLUDED.source_format,
			source_url = EXCLUDED.source_url,
			first_seen_at = LEAST(goreecloud_feeds.feeds.first_seen_at, EXCLUDED.first_seen_at),
			last_seen_at = GREATEST(goreecloud_feeds.feeds.last_seen_at, EXCLUDED.last_seen_at),
			updated_at = CURRENT_TIMESTAMP
	`,
		input.Feed.ID,
		input.Feed.URL,
		input.NormalizedURL,
		input.Feed.Title,
		input.Feed.Description,
		input.Feed.IconURL,
		input.Feed.ImageURL,
		input.Feed.Language,
		input.Feed.Source.Format,
		input.Feed.Source.URL,
		input.SeenAt.UTC(),
	); err != nil {
		return fmt.Errorf("upsert feed: %w", err)
	}
	return nil
}

func (s *Store) UpsertSubscription(ctx context.Context, input SubscriptionWrite) error {
	if err := ensureStore(s); err != nil {
		return err
	}
	subscription := input.Subscription
	if strings.TrimSpace(string(subscription.ID)) == "" {
		return fmt.Errorf("subscription id is required")
	}
	if strings.TrimSpace(string(subscription.UserID)) == "" || strings.TrimSpace(string(subscription.FeedID)) == "" {
		return fmt.Errorf("subscription user id and feed id are required")
	}
	if subscription.CreatedAt.IsZero() {
		return fmt.Errorf("subscription creation time is required")
	}

	var storedUserID feed.ID
	var storedFeedID feed.ID
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO goreecloud_feeds.subscriptions (
			id, user_id, feed_id, created_at, updated_at, disabled
		)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, $5)
		ON CONFLICT (id) DO UPDATE SET
			disabled = EXCLUDED.disabled,
			updated_at = CURRENT_TIMESTAMP
		RETURNING user_id, feed_id
	`,
		subscription.ID,
		subscription.UserID,
		subscription.FeedID,
		subscription.CreatedAt.UTC(),
		input.Disabled,
	).Scan(&storedUserID, &storedFeedID); err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	if storedUserID != subscription.UserID || storedFeedID != subscription.FeedID {
		return fmt.Errorf("subscription id %q is already bound to a different user/feed pair", subscription.ID)
	}
	return nil
}

func (s *Store) UpsertArticle(ctx context.Context, input ArticleWrite) error {
	if err := ensureStore(s); err != nil {
		return err
	}
	if err := validateArticleWrite(input); err != nil {
		return err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin article transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var storedFeedID feed.ID
	if err := tx.QueryRow(ctx, `
		INSERT INTO goreecloud_feeds.articles (
			id, feed_id, source_identifier, canonical_url, normalized_url,
			title, author, published_at, source_updated_at, summary, content,
			language, source_format, source_url, content_fingerprint,
			first_retrieved_at, last_retrieved_at, retention_eligible_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $16, $17,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		ON CONFLICT (id) DO UPDATE SET
			source_identifier = EXCLUDED.source_identifier,
			canonical_url = EXCLUDED.canonical_url,
			normalized_url = EXCLUDED.normalized_url,
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			published_at = EXCLUDED.published_at,
			source_updated_at = EXCLUDED.source_updated_at,
			summary = EXCLUDED.summary,
			content = EXCLUDED.content,
			language = EXCLUDED.language,
			source_format = EXCLUDED.source_format,
			source_url = EXCLUDED.source_url,
			content_fingerprint = EXCLUDED.content_fingerprint,
			first_retrieved_at = LEAST(goreecloud_feeds.articles.first_retrieved_at, EXCLUDED.first_retrieved_at),
			last_retrieved_at = GREATEST(goreecloud_feeds.articles.last_retrieved_at, EXCLUDED.last_retrieved_at),
			retention_eligible_at = EXCLUDED.retention_eligible_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING feed_id
	`,
		input.Article.ID,
		input.Article.FeedID,
		input.Article.Identifier,
		input.Article.URL,
		input.NormalizedURL,
		input.Article.Title,
		input.Article.Author,
		input.Article.PublishedAt,
		input.Article.UpdatedAt,
		input.Article.Summary,
		input.Article.Content,
		input.Article.Language,
		input.Article.Source.Format,
		input.Article.Source.URL,
		input.ContentFingerprint,
		input.RetrievedAt.UTC(),
		input.RetentionEligibleAt,
	).Scan(&storedFeedID); err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}
	if storedFeedID != input.Article.FeedID {
		return fmt.Errorf("article id %q is already bound to a different feed", input.Article.ID)
	}

	for _, key := range input.IdentityKeys {
		if err := ensureIdentityKey(ctx, tx, input.Article.FeedID, input.Article.ID, key); err != nil {
			return err
		}
	}

	if err := ensureSourceHistory(ctx, tx, input); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit article transaction: %w", err)
	}
	return nil
}

func (s *Store) ArticleByID(ctx context.Context, articleID feed.ID) (StoredArticle, error) {
	if err := ensureStore(s); err != nil {
		return StoredArticle{}, err
	}
	if strings.TrimSpace(string(articleID)) == "" {
		return StoredArticle{}, fmt.Errorf("article id is required")
	}

	var out StoredArticle
	out.Article.ID = articleID
	if err := s.pool.QueryRow(ctx, `
		SELECT
			feed_id, source_identifier, canonical_url, normalized_url, title,
			author, published_at, source_updated_at, summary, content, language,
			source_format, source_url, content_fingerprint, first_retrieved_at,
			last_retrieved_at, retention_eligible_at
		FROM goreecloud_feeds.articles
		WHERE id = $1
	`, articleID).Scan(
		&out.Article.FeedID,
		&out.Article.Identifier,
		&out.Article.URL,
		&out.NormalizedURL,
		&out.Article.Title,
		&out.Article.Author,
		&out.Article.PublishedAt,
		&out.Article.UpdatedAt,
		&out.Article.Summary,
		&out.Article.Content,
		&out.Article.Language,
		&out.Article.Source.Format,
		&out.Article.Source.URL,
		&out.ContentFingerprint,
		&out.FirstRetrievedAt,
		&out.LastRetrievedAt,
		&out.RetentionEligibleAt,
	); err != nil {
		return StoredArticle{}, fmt.Errorf("read article %q: %w", articleID, err)
	}
	return out, nil
}

func (s *Store) UpsertArticleState(ctx context.Context, input ArticleStateWrite) error {
	if err := ensureStore(s); err != nil {
		return err
	}
	state := input.State
	if strings.TrimSpace(string(state.UserID)) == "" || strings.TrimSpace(string(state.ArticleID)) == "" {
		return fmt.Errorf("article state user id and article id are required")
	}
	if state.ReadPosition < 0 || state.ReadPosition > 1 {
		return fmt.Errorf("article read position must be between 0 and 1")
	}
	if state.UpdatedAt.IsZero() {
		return fmt.Errorf("article state update time is required")
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO goreecloud_feeds.article_states (
			user_id, article_id, read, saved, favorite, preserved,
			read_position, last_read_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, article_id) DO UPDATE SET
			read = EXCLUDED.read,
			saved = EXCLUDED.saved,
			favorite = EXCLUDED.favorite,
			preserved = EXCLUDED.preserved,
			read_position = EXCLUDED.read_position,
			last_read_at = EXCLUDED.last_read_at,
			updated_at = EXCLUDED.updated_at
	`,
		state.UserID,
		state.ArticleID,
		state.Read,
		state.Saved,
		state.Favorite,
		input.Preserved,
		state.ReadPosition,
		input.LastReadAt,
		state.UpdatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("upsert article state: %w", err)
	}
	return nil
}

func (s *Store) ArticleStateForUser(ctx context.Context, userID, articleID feed.ID) (StoredArticleState, error) {
	if err := ensureStore(s); err != nil {
		return StoredArticleState{}, err
	}
	if strings.TrimSpace(string(userID)) == "" || strings.TrimSpace(string(articleID)) == "" {
		return StoredArticleState{}, fmt.Errorf("article state user id and article id are required")
	}

	var out StoredArticleState
	out.State.UserID = userID
	out.State.ArticleID = articleID
	if err := s.pool.QueryRow(ctx, `
		SELECT read, saved, favorite, preserved, read_position, last_read_at, updated_at
		FROM goreecloud_feeds.article_states
		WHERE user_id = $1 AND article_id = $2
	`, userID, articleID).Scan(
		&out.State.Read,
		&out.State.Saved,
		&out.State.Favorite,
		&out.Preserved,
		&out.State.ReadPosition,
		&out.LastReadAt,
		&out.State.UpdatedAt,
	); err != nil {
		return StoredArticleState{}, fmt.Errorf("read article state: %w", err)
	}
	return out, nil
}

func validateArticleWrite(input ArticleWrite) error {
	if strings.TrimSpace(string(input.Article.ID)) == "" {
		return fmt.Errorf("article id is required")
	}
	if strings.TrimSpace(string(input.Article.FeedID)) == "" {
		return fmt.Errorf("article feed id is required")
	}
	if input.RetrievedAt.IsZero() {
		return fmt.Errorf("article retrieval time is required")
	}
	if strings.TrimSpace(string(input.SourceHistoryID)) == "" {
		return fmt.Errorf("article source history id is required")
	}
	if len(input.Article.Categories) > 0 || len(input.Article.Tags) > 0 || len(input.Article.Media) > 0 || len(input.Article.Images) > 0 {
		return ErrUnsupportedArticleMetadata
	}
	for _, key := range input.IdentityKeys {
		if err := validateIdentityKey(key); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentityKey(key ArticleIdentityKey) error {
	switch key.Type {
	case feed.DuplicateBySourceIdentifier, feed.DuplicateByNormalizedURL, feed.DuplicateByContentFingerprint:
	default:
		return fmt.Errorf("unsupported article identity key type %q", key.Type)
	}
	if strings.TrimSpace(key.Value) == "" {
		return fmt.Errorf("article identity key value is required")
	}
	return nil
}

func ensureIdentityKey(
	ctx context.Context,
	tx pgx.Tx,
	feedID feed.ID,
	articleID feed.ID,
	key ArticleIdentityKey,
) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO goreecloud_feeds.article_identity_keys (
			feed_id, key_type, key_value, article_id
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (feed_id, key_type, key_value) DO NOTHING
	`, feedID, key.Type, key.Value, articleID); err != nil {
		return fmt.Errorf("store article identity key: %w", err)
	}

	var storedArticleID feed.ID
	if err := tx.QueryRow(ctx, `
		SELECT article_id
		FROM goreecloud_feeds.article_identity_keys
		WHERE feed_id = $1 AND key_type = $2 AND key_value = $3
	`, feedID, key.Type, key.Value).Scan(&storedArticleID); err != nil {
		return fmt.Errorf("verify article identity key: %w", err)
	}
	if storedArticleID != articleID {
		return fmt.Errorf(
			"article identity key %q/%q is already bound to article %q",
			key.Type,
			key.Value,
			storedArticleID,
		)
	}
	return nil
}

func ensureSourceHistory(ctx context.Context, tx pgx.Tx, input ArticleWrite) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO goreecloud_feeds.article_source_history (
			id, article_id, feed_id, retrieved_at, source_identifier,
			canonical_url, normalized_url, title, author, published_at,
			source_updated_at, content_fingerprint
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
	`,
		input.SourceHistoryID,
		input.Article.ID,
		input.Article.FeedID,
		input.RetrievedAt.UTC(),
		input.Article.Identifier,
		input.Article.URL,
		input.NormalizedURL,
		input.Article.Title,
		input.Article.Author,
		input.Article.PublishedAt,
		input.Article.UpdatedAt,
		input.ContentFingerprint,
	); err != nil {
		return fmt.Errorf("store article source history: %w", err)
	}

	var storedArticleID feed.ID
	var storedFeedID feed.ID
	if err := tx.QueryRow(ctx, `
		SELECT article_id, feed_id
		FROM goreecloud_feeds.article_source_history
		WHERE id = $1
	`, input.SourceHistoryID).Scan(&storedArticleID, &storedFeedID); err != nil {
		return fmt.Errorf("verify article source history: %w", err)
	}
	if storedArticleID != input.Article.ID || storedFeedID != input.Article.FeedID {
		return fmt.Errorf("article source history id %q is already bound to a different article/feed", input.SourceHistoryID)
	}
	return nil
}

func ensureStore(s *Store) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("postgres store is not open")
	}
	return nil
}
