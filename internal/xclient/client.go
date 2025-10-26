package xclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"starseed/internal/model"
)

// XClient defines methods we use from X API.
type XClient interface {
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error)
	GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error)
}

// HTTPClient is a simple bearer-token client for X API v2.
type HTTPClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

func NewHTTPClient(bearerToken string) *HTTPClient {
	return &HTTPClient{
        baseURL:     "https://api.twitter.com/2",
		bearerToken: bearerToken,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
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
	resp, err := c.httpClient.Do(req)
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

func (c *HTTPClient) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) {
	u := fmt.Sprintf("%s/users/%s/following?max_results=%d&user.fields=public_metrics,created_at,verified,description,url,profile_image_url", c.baseURL, url.PathEscape(userID), clamp(limit, 10, 1000))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	c.auth(req)
	resp, err := c.httpClient.Do(req)
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

func clamp(v, min, max int) int { if v < min { return min }; if v > max { return max }; return v }
