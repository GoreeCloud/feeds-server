package feed

import "time"

// ID is an internal GoreeCloud Feeds identifier. Persistence-backed ID
// generation is intentionally outside the parser/model tranche.
type ID string

// SourceInfo records where normalized feed data came from.
type SourceInfo struct {
	Format string
	URL    string
}

// User is the internal identity reference for user-owned Feeds state.
// Authentication and GoreeCloud Identity integration are separate concerns.
type User struct {
	ID ID
}

// Subscription connects one user to one feed without making feed content
// user-specific.
type Subscription struct {
	ID        ID
	UserID    ID
	FeedID    ID
	FeedURL   string
	CreatedAt time.Time
}

// Feed contains feed-owned data that may be shared by many users.
type Feed struct {
	ID          ID
	URL         string
	Title       string
	Description string
	IconURL     string
	ImageURL    string
	Language    string
	Source      SourceInfo
}

// Media describes externally referenced media associated with an article.
type Media struct {
	URL      string
	MIMEType string
	Width    int
	Height   int
}

// Image describes an externally referenced image associated with an article.
type Image struct {
	URL    string
	Width  int
	Height int
}

// Article contains feed-owned normalized article data.
type Article struct {
	ID          ID
	FeedID      ID
	Identifier  string
	URL         string
	Title       string
	Author      string
	PublishedAt *time.Time
	UpdatedAt   *time.Time
	Summary     string
	Content     string
	Categories  []string
	Tags        []string
	Media       []Media
	Images      []Image
	Language    string
	Source      SourceInfo
}

// ArticleState is deliberately separate from Article so multiple users can
// consume the same feed/article content while retaining independent state.
type ArticleState struct {
	UserID       ID
	ArticleID    ID
	Read         bool
	Saved        bool
	Favorite     bool
	ReadPosition float64
	UpdatedAt    time.Time
}

// ParsedFeed is the normalized result of parsing one feed document.
// Warnings describe recoverable omissions without converting usable input
// into a total failure.
type ParsedFeed struct {
	Feed     Feed
	Articles []Article
	Warnings []string
}
