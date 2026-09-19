BEGIN;

CREATE TABLE feeds (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    url text NOT NULL UNIQUE CHECK (length(btrim(url)) > 0),
    title text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    icon_url text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT '',
    source_format text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    user_id text NOT NULL CHECK (length(btrim(user_id)) > 0),
    feed_id text NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    feed_url text NOT NULL CHECK (length(btrim(feed_url)) > 0),
    retention_days integer CHECK (retention_days IS NULL OR retention_days > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, feed_id)
);

CREATE INDEX subscriptions_user_id_idx ON subscriptions (user_id);
CREATE INDEX subscriptions_feed_id_idx ON subscriptions (feed_id);

CREATE TABLE articles (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    feed_id text NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    identifier text NOT NULL DEFAULT '',
    url text NOT NULL DEFAULT '',
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
    first_retrieved_at timestamptz NOT NULL,
    last_retrieved_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (last_retrieved_at >= first_retrieved_at)
);

CREATE INDEX articles_feed_id_idx ON articles (feed_id);
CREATE INDEX articles_published_at_idx ON articles (published_at DESC);
CREATE INDEX articles_last_retrieved_at_idx ON articles (last_retrieved_at DESC);
CREATE INDEX articles_content_fingerprint_idx
    ON articles (feed_id, content_fingerprint)
    WHERE content_fingerprint <> '';

CREATE TABLE article_aliases (
    feed_id text NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    alias_type text NOT NULL
        CHECK (alias_type IN ('source_identifier', 'normalized_url', 'content_fingerprint')),
    alias_value text NOT NULL CHECK (length(btrim(alias_value)) > 0),
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    PRIMARY KEY (feed_id, alias_type, alias_value),
    CHECK (last_seen_at >= first_seen_at)
);

CREATE INDEX article_aliases_article_id_idx ON article_aliases (article_id);

CREATE TABLE article_source_history (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    retrieved_at timestamptz NOT NULL,
    source_identifier text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '',
    source_format text NOT NULL DEFAULT '',
    original_content text NOT NULL DEFAULT '',
    content_fingerprint text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX article_source_history_article_id_idx
    ON article_source_history (article_id, retrieved_at DESC);

CREATE TABLE article_states (
    user_id text NOT NULL CHECK (length(btrim(user_id)) > 0),
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    read boolean NOT NULL DEFAULT false,
    saved boolean NOT NULL DEFAULT false,
    favorite boolean NOT NULL DEFAULT false,
    preserved_at timestamptz,
    read_position double precision NOT NULL DEFAULT 0 CHECK (read_position >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, article_id)
);

CREATE INDEX article_states_article_id_idx ON article_states (article_id);
CREATE INDEX article_states_saved_idx
    ON article_states (user_id, saved)
    WHERE saved = true;
CREATE INDEX article_states_favorite_idx
    ON article_states (user_id, favorite)
    WHERE favorite = true;
CREATE INDEX article_states_preserved_idx
    ON article_states (user_id, preserved_at)
    WHERE preserved_at IS NOT NULL;

CREATE TABLE article_categories (
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    category text NOT NULL CHECK (length(btrim(category)) > 0),
    PRIMARY KEY (article_id, category)
);

CREATE TABLE article_tags (
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag text NOT NULL CHECK (length(btrim(tag)) > 0),
    PRIMARY KEY (article_id, tag)
);

CREATE TABLE article_media (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    url text NOT NULL CHECK (length(btrim(url)) > 0),
    mime_type text NOT NULL DEFAULT '',
    width integer NOT NULL DEFAULT 0 CHECK (width >= 0),
    height integer NOT NULL DEFAULT 0 CHECK (height >= 0),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    UNIQUE (article_id, position)
);

CREATE INDEX article_media_article_id_idx ON article_media (article_id);

CREATE TABLE article_images (
    id text PRIMARY KEY CHECK (length(btrim(id)) > 0),
    article_id text NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    url text NOT NULL CHECK (length(btrim(url)) > 0),
    width integer NOT NULL DEFAULT 0 CHECK (width >= 0),
    height integer NOT NULL DEFAULT 0 CHECK (height >= 0),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    UNIQUE (article_id, position)
);

CREATE INDEX article_images_article_id_idx ON article_images (article_id);

COMMIT;
