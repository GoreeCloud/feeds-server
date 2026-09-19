BEGIN;

CREATE SCHEMA IF NOT EXISTS goreecloud_feeds;

CREATE TABLE goreecloud_feeds.users (
    id text PRIMARY KEY,
    identity_subject text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT users_identity_subject_not_blank CHECK (btrim(identity_subject) <> '')
);

CREATE TABLE goreecloud_feeds.feeds (
    id text PRIMARY KEY,
    canonical_url text NOT NULL,
    normalized_url text NOT NULL UNIQUE,
    title text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    icon_url text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT '',
    source_format text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '',
    first_seen_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT feeds_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT feeds_canonical_url_not_blank CHECK (btrim(canonical_url) <> ''),
    CONSTRAINT feeds_normalized_url_not_blank CHECK (btrim(normalized_url) <> '')
);

CREATE TABLE goreecloud_feeds.subscriptions (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES goreecloud_feeds.users(id) ON DELETE CASCADE,
    feed_id text NOT NULL REFERENCES goreecloud_feeds.feeds(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled boolean NOT NULL DEFAULT false,
    CONSTRAINT subscriptions_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT subscriptions_user_feed_unique UNIQUE (user_id, feed_id)
);

CREATE TABLE goreecloud_feeds.articles (
    id text PRIMARY KEY,
    feed_id text NOT NULL REFERENCES goreecloud_feeds.feeds(id) ON DELETE RESTRICT,
    source_identifier text NOT NULL DEFAULT '',
    canonical_url text NOT NULL DEFAULT '',
    normalized_url text NOT NULL DEFAULT '',
    title text NOT NULL DEFAULT '',
    author text NOT NULL DEFAULT '',
    published_at timestamptz,
    source_updated_at timestamptz,
    summary text NOT NULL DEFAULT '',
    content text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT '',
    source_format text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '',
    content_fingerprint text NOT NULL DEFAULT '',
    first_retrieved_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_retrieved_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retention_eligible_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT articles_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT articles_id_feed_unique UNIQUE (id, feed_id),
    CONSTRAINT articles_content_fingerprint_sha256_hex CHECK (
        content_fingerprint = ''
        OR content_fingerprint ~ '^[0-9a-f]{64}$'
    )
);

CREATE TABLE goreecloud_feeds.article_identity_keys (
    feed_id text NOT NULL,
    key_type text NOT NULL,
    key_value text NOT NULL,
    article_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (feed_id, key_type, key_value),
    CONSTRAINT article_identity_keys_article_feed_fk
        FOREIGN KEY (article_id, feed_id)
        REFERENCES goreecloud_feeds.articles(id, feed_id)
        ON DELETE CASCADE,
    CONSTRAINT article_identity_keys_type_valid CHECK (
        key_type IN ('source_identifier', 'normalized_url', 'content_fingerprint')
    ),
    CONSTRAINT article_identity_keys_value_not_blank CHECK (btrim(key_value) <> '')
);

CREATE TABLE goreecloud_feeds.article_source_history (
    id text PRIMARY KEY,
    article_id text NOT NULL,
    feed_id text NOT NULL,
    retrieved_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source_identifier text NOT NULL DEFAULT '',
    canonical_url text NOT NULL DEFAULT '',
    normalized_url text NOT NULL DEFAULT '',
    title text NOT NULL DEFAULT '',
    author text NOT NULL DEFAULT '',
    published_at timestamptz,
    source_updated_at timestamptz,
    content_fingerprint text NOT NULL DEFAULT '',
    CONSTRAINT article_source_history_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT article_source_history_article_feed_fk
        FOREIGN KEY (article_id, feed_id)
        REFERENCES goreecloud_feeds.articles(id, feed_id)
        ON DELETE CASCADE,
    CONSTRAINT article_source_history_content_fingerprint_sha256_hex CHECK (
        content_fingerprint = ''
        OR content_fingerprint ~ '^[0-9a-f]{64}$'
    )
);

CREATE TABLE goreecloud_feeds.article_states (
    user_id text NOT NULL REFERENCES goreecloud_feeds.users(id) ON DELETE CASCADE,
    article_id text NOT NULL REFERENCES goreecloud_feeds.articles(id) ON DELETE CASCADE,
    read boolean NOT NULL DEFAULT false,
    saved boolean NOT NULL DEFAULT false,
    favorite boolean NOT NULL DEFAULT false,
    preserved boolean NOT NULL DEFAULT false,
    read_position double precision NOT NULL DEFAULT 0,
    last_read_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, article_id),
    CONSTRAINT article_states_read_position_range CHECK (
        read_position >= 0 AND read_position <= 1
    )
);

CREATE INDEX subscriptions_feed_id_idx
    ON goreecloud_feeds.subscriptions(feed_id);

CREATE INDEX articles_feed_published_idx
    ON goreecloud_feeds.articles(feed_id, published_at DESC);

CREATE INDEX articles_retention_eligible_idx
    ON goreecloud_feeds.articles(retention_eligible_at)
    WHERE retention_eligible_at IS NOT NULL;

CREATE INDEX article_identity_keys_article_id_idx
    ON goreecloud_feeds.article_identity_keys(article_id);

CREATE INDEX article_source_history_article_retrieved_idx
    ON goreecloud_feeds.article_source_history(article_id, retrieved_at DESC);

CREATE INDEX article_states_article_id_idx
    ON goreecloud_feeds.article_states(article_id);

CREATE INDEX article_states_user_unread_idx
    ON goreecloud_feeds.article_states(user_id, article_id)
    WHERE read = false;

CREATE INDEX article_states_user_saved_idx
    ON goreecloud_feeds.article_states(user_id, article_id)
    WHERE saved = true OR favorite = true OR preserved = true;

COMMIT;
