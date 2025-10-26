package xclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
    "os"
    "strconv"
	"time"

	"starseed/internal/model"
    "golang.org/x/time/rate"
    "strings"
)

// XClient defines methods we use from X API.
type XClient interface {
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error)
	GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error)
    SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error)
    GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error)
    GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error)
    GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error)
    GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error)
}

// HTTPClient is a simple bearer-token client for X API v2.
type HTTPClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
    limiter     *rate.Limiter
    maxAttempts int
    baseBackoff time.Duration
}

func NewHTTPClient(bearerToken string) *HTTPClient {
	return &HTTPClient{
        baseURL:     "https://api.twitter.com/2",
		bearerToken: bearerToken,
        httpClient:  &http.Client{Timeout: 15 * time.Second},
        limiter:     newDefaultLimiter(),
        maxAttempts: getEnvInt("X_API_MAX_ATTEMPTS", 5),
        baseBackoff: time.Duration(getEnvInt("X_API_BASE_BACKOFF_MS", 500)) * time.Millisecond,
	}
}

func (c *HTTPClient) auth(req *http.Request) {
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	req.Header.Set("Accept", "application/json")
}

func (c *HTTPClient) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	var out model.User
	if username == "" {
		return out, errors.New("empty username")
	}
	u := fmt.Sprintf("%s/users/by/username/%s?user.fields=public_metrics,created_at,verified,description,url,profile_image_url", c.baseURL, url.PathEscape(username))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return out, err }
    resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return out, fmt.Errorf("x api status %d", resp.StatusCode)
	}
	var raw struct {
		Data struct {
			ID       string    `json:"id"`
			Name     string    `json:"name"`
			Username string    `json:"username"`
			CreatedAt time.Time `json:"created_at"`
			Verified bool      `json:"verified"`
			Description string `json:"description"`
			URL string `json:"url"`
			PublicMetrics struct {
				FollowersCount int `json:"followers_count"`
				FollowingCount int `json:"following_count"`
				TweetCount     int `json:"tweet_count"`
				ListedCount    int `json:"listed_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return out, err
	}
	out = model.User{
		ID: raw.Data.ID,
		Username: raw.Data.Username,
		Name: raw.Data.Name,
		CreatedAt: raw.Data.CreatedAt,
		Verified: raw.Data.Verified,
		Description: raw.Data.Description,
		URL: raw.Data.URL,
		FollowersCount: raw.Data.PublicMetrics.FollowersCount,
		FollowingCount: raw.Data.PublicMetrics.FollowingCount,
		TweetCount: raw.Data.PublicMetrics.TweetCount,
		ListedCount: raw.Data.PublicMetrics.ListedCount,
	}
	return out, nil
}

func (c *HTTPClient) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
	// Placeholder: timeline endpoint may differ; map response if available.
	_ = userID
	_ = limit
	return []model.Tweet{}, nil
}

// SearchRecentTweets searches recent tweets by query of interests.
func (c *HTTPClient) SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error) {
    u := fmt.Sprintf("%s/tweets/search/recent?max_results=%d&tweet.fields=created_at,public_metrics,lang,author_id&query=%s",
        c.baseURL, clamp(limit, 10, 100), url.QueryEscape(query))
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return nil, fmt.Errorf("x api status %d", resp.StatusCode) }
    var raw struct {
        Data []struct{
            ID string `json:"id"`
            Text string `json:"text"`
            CreatedAt time.Time `json:"created_at"`
            Lang string `json:"lang"`
            AuthorID string `json:"author_id"`
            PublicMetrics struct{
                LikeCount int `json:"like_count"`
                ReplyCount int `json:"reply_count"`
                RetweetCount int `json:"retweet_count"`
                QuoteCount int `json:"quote_count"`
            } `json:"public_metrics"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    out := make([]model.Tweet, 0, len(raw.Data))
    for _, d := range raw.Data {
        out = append(out, model.Tweet{
            ID: d.ID,
            Text: d.Text,
            CreatedAt: d.CreatedAt,
            Language: d.Lang,
            AuthorID: d.AuthorID,
            LikeCount: d.PublicMetrics.LikeCount,
            ReplyCount: d.PublicMetrics.ReplyCount,
            RetweetCount: d.PublicMetrics.RetweetCount,
            QuoteCount: d.PublicMetrics.QuoteCount,
        })
    }
    return out, nil
}

func (c *HTTPClient) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) {
	u := fmt.Sprintf("%s/users/%s/following?max_results=%d&user.fields=public_metrics,created_at,verified,description,url,profile_image_url", c.baseURL, url.PathEscape(userID), clamp(limit, 10, 1000))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("x api status %d", resp.StatusCode)
	}
	var raw struct {
		Data []struct {
			ID       string    `json:"id"`
			Name     string    `json:"name"`
			Username string    `json:"username"`
			CreatedAt time.Time `json:"created_at"`
			Verified bool      `json:"verified"`
			Description string `json:"description"`
			URL string `json:"url"`
			PublicMetrics struct {
				FollowersCount int `json:"followers_count"`
				FollowingCount int `json:"following_count"`
				TweetCount     int `json:"tweet_count"`
				ListedCount    int `json:"listed_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]model.User, 0, len(raw.Data))
	for _, d := range raw.Data {
		out = append(out, model.User{
			ID: d.ID,
			Username: d.Username,
			Name: d.Name,
			CreatedAt: d.CreatedAt,
			Verified: d.Verified,
			Description: d.Description,
			URL: d.URL,
			FollowersCount: d.PublicMetrics.FollowersCount,
			FollowingCount: d.PublicMetrics.FollowingCount,
			TweetCount: d.PublicMetrics.TweetCount,
			ListedCount: d.PublicMetrics.ListedCount,
		})
	}
	return out, nil
}

// GetUserTweets returns recent tweets for a user.
func (c *HTTPClient) GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
    u := fmt.Sprintf("%s/users/%s/tweets?max_results=%d&tweet.fields=created_at,public_metrics,lang&exclude=retweets,replies",
        c.baseURL, url.PathEscape(userID), clamp(limit, 5, 100))
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return nil, fmt.Errorf("x api status %d", resp.StatusCode) }
    var raw struct {
        Data []struct{
            ID string `json:"id"`
            Text string `json:"text"`
            CreatedAt time.Time `json:"created_at"`
            Lang string `json:"lang"`
            PublicMetrics struct{
                LikeCount int `json:"like_count"`
                ReplyCount int `json:"reply_count"`
                RetweetCount int `json:"retweet_count"`
                QuoteCount int `json:"quote_count"`
            } `json:"public_metrics"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    out := make([]model.Tweet, 0, len(raw.Data))
    for _, d := range raw.Data {
        out = append(out, model.Tweet{
            ID: d.ID,
            AuthorID: userID,
            Text: d.Text,
            CreatedAt: d.CreatedAt,
            Language: d.Lang,
            LikeCount: d.PublicMetrics.LikeCount,
            ReplyCount: d.PublicMetrics.ReplyCount,
            RetweetCount: d.PublicMetrics.RetweetCount,
            QuoteCount: d.PublicMetrics.QuoteCount,
        })
    }
    return out, nil
}

// GetUsersByIDs fetches user objects for given ids in one request.
func (c *HTTPClient) GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error) {
    if len(ids) == 0 { return nil, nil }
    // Join up to 100 IDs as allowed by API
    if len(ids) > 100 { ids = ids[:100] }
    joined := url.QueryEscape(strings.Join(ids, ","))
    u := fmt.Sprintf("%s/users?ids=%s&user.fields=public_metrics,created_at,verified,description,url,profile_image_url", c.baseURL, joined)
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return nil, fmt.Errorf("x api status %d", resp.StatusCode) }
    var raw struct {
        Data []struct {
            ID       string    `json:"id"`
            Name     string    `json:"name"`
            Username string    `json:"username"`
            CreatedAt time.Time `json:"created_at"`
            Verified bool      `json:"verified"`
            Description string `json:"description"`
            URL string `json:"url"`
            PublicMetrics struct {
                FollowersCount int `json:"followers_count"`
                FollowingCount int `json:"following_count"`
                TweetCount     int `json:"tweet_count"`
                ListedCount    int `json:"listed_count"`
            } `json:"public_metrics"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    out := make([]model.User, 0, len(raw.Data))
    for _, d := range raw.Data {
        out = append(out, model.User{
            ID: d.ID,
            Username: d.Username,
            Name: d.Name,
            CreatedAt: d.CreatedAt,
            Verified: d.Verified,
            Description: d.Description,
            URL: d.URL,
            FollowersCount: d.PublicMetrics.FollowersCount,
            FollowingCount: d.PublicMetrics.FollowingCount,
            TweetCount: d.PublicMetrics.TweetCount,
            ListedCount: d.PublicMetrics.ListedCount,
        })
    }
    return out, nil
}

// GetLikedTweets returns tweets liked by the user.
func (c *HTTPClient) GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
    u := fmt.Sprintf("%s/users/%s/liked_tweets?max_results=%d&tweet.fields=created_at,public_metrics,lang,author_id",
        c.baseURL, url.PathEscape(userID), clamp(limit, 10, 100))
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return nil, fmt.Errorf("x api status %d", resp.StatusCode) }
    var raw struct {
        Data []struct {
            ID string `json:"id"`
            Text string `json:"text"`
            AuthorID string `json:"author_id"`
            CreatedAt time.Time `json:"created_at"`
            Lang string `json:"lang"`
            PublicMetrics struct{
                LikeCount int `json:"like_count"`
                ReplyCount int `json:"reply_count"`
                RetweetCount int `json:"retweet_count"`
                QuoteCount int `json:"quote_count"`
            } `json:"public_metrics"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    out := make([]model.Tweet, 0, len(raw.Data))
    for _, d := range raw.Data {
        out = append(out, model.Tweet{
            ID: d.ID,
            AuthorID: d.AuthorID,
            Text: d.Text,
            CreatedAt: d.CreatedAt,
            Language: d.Lang,
            LikeCount: d.PublicMetrics.LikeCount,
            ReplyCount: d.PublicMetrics.ReplyCount,
            RetweetCount: d.PublicMetrics.RetweetCount,
            QuoteCount: d.PublicMetrics.QuoteCount,
        })
    }
    return out, nil
}

// GetMentions returns tweets that mention the user.
func (c *HTTPClient) GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
    u := fmt.Sprintf("%s/users/%s/mentions?max_results=%d&tweet.fields=created_at,public_metrics,lang,author_id",
        c.baseURL, url.PathEscape(userID), clamp(limit, 10, 100))
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    c.auth(req)
    if err := c.limiter.Wait(ctx); err != nil { return nil, err }
    resp, err := c.doWithRetry(ctx, req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return nil, fmt.Errorf("x api status %d", resp.StatusCode) }
    var raw struct {
        Data []struct {
            ID string `json:"id"`
            Text string `json:"text"`
            AuthorID string `json:"author_id"`
            CreatedAt time.Time `json:"created_at"`
            Lang string `json:"lang"`
            PublicMetrics struct{
                LikeCount int `json:"like_count"`
                ReplyCount int `json:"reply_count"`
                RetweetCount int `json:"retweet_count"`
                QuoteCount int `json:"quote_count"`
            } `json:"public_metrics"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    out := make([]model.Tweet, 0, len(raw.Data))
    for _, d := range raw.Data {
        out = append(out, model.Tweet{
            ID: d.ID,
            AuthorID: d.AuthorID,
            Text: d.Text,
            CreatedAt: d.CreatedAt,
            Language: d.Lang,
            LikeCount: d.PublicMetrics.LikeCount,
            ReplyCount: d.PublicMetrics.ReplyCount,
            RetweetCount: d.PublicMetrics.RetweetCount,
            QuoteCount: d.PublicMetrics.QuoteCount,
        })
    }
    return out, nil
}

func clamp(v, min, max int) int { if v < min { return min }; if v > max { return max }; return v }

func (c *HTTPClient) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
    backoff := c.baseBackoff
    var lastErr error
    for attempt := 1; attempt <= c.maxAttempts; attempt++ {
        resp, err := c.httpClient.Do(req.Clone(ctx))
        if err == nil {
            if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
                ra := resp.Header.Get("Retry-After")
                _ = resp.Body.Close()
                wait := backoff
                if ra != "" {
                    if secs, err := strconv.Atoi(ra); err == nil {
                        wait = time.Duration(secs) * time.Second
                    } else if t, err := http.ParseTime(ra); err == nil {
                        if d := time.Until(t); d > 0 { wait = d }
                    }
                }
                // jitter +/-20%
                jitter := time.Duration(float64(wait) * 0.2)
                if jitter > 0 {
                    wait = wait - jitter + time.Duration(time.Now().UnixNano()%int64(2*jitter))
                }
                select {
                case <-time.After(wait):
                case <-ctx.Done():
                    return nil, ctx.Err()
                }
                backoff *= 2
                continue
            }
            return resp, nil
        }
        lastErr = err
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
        backoff *= 2
    }
    return nil, fmt.Errorf("request failed after %d attempts: %v", c.maxAttempts, lastErr)
}

func getEnvInt(key string, def int) int {
    v := os.Getenv(key)
    if v == "" { return def }
    if i, err := strconv.Atoi(v); err == nil && i > 0 { return i }
    return def
}
