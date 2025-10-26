package model

import "time"

// User represents a subset of X user fields used by the tool.
type User struct {
	ID              string
	Username        string
	Name            string
	Description     string
	CreatedAt       time.Time
	FollowersCount  int
	FollowingCount  int
	TweetCount      int
	ListedCount     int
	DefaultProfile  bool
	DefaultImage    bool
	Verified        bool
	URL             string
	Language        string
}

// Tweet represents a subset of X tweet fields used by the tool.
type Tweet struct {
	ID        string
	AuthorID  string
	Text      string
	CreatedAt time.Time
	LikeCount int
	ReplyCount int
	RetweetCount int
	QuoteCount int
	Language  string
	HasLink   bool
}

// EngagementEvent captures an engagement we did or received.
type EngagementEvent struct {
	Timestamp   time.Time
	Type        string // like, reply, retweet, follow
	TargetTweet string // tweet id if applicable
	TargetUser  string // user id if applicable
}
