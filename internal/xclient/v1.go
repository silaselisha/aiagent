package xclient

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"starseed/internal/model"
)

// V1Client supports X API v1.1 home timeline via OAuth 1.0a.
type V1Client struct {
	Base           *HTTPClient
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
	nowFn   func() time.Time
	nonceFn func() string
}

func NewV1Client(base *HTTPClient, ck, cs, at, as string) *V1Client {
	return &V1Client{
		Base:           base,
		ConsumerKey:    ck,
		ConsumerSecret: cs,
		AccessToken:    at,
		AccessSecret:   as,
		nowFn:          time.Now,
		nonceFn:        func() string { return strconv.FormatInt(rand.Int63(), 36) },
	}
}

// GetHomeTimeline returns recent tweets from the user's home timeline.
func (c *V1Client) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
	endpoint := "https://api.twitter.com/1.1/statuses/home_timeline.json"
	params := map[string]string{
		"count":      fmt.Sprintf("%d", clamp(limit, 5, 200)),
		"tweet_mode": "extended",
	}
	reqURL := endpoint + "?" + encodeQuery(params)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	c.oauth1Sign(req, params)
	if err := c.Base.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	resp, err := c.Base.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("x v1 status %d", resp.StatusCode)
	}
	var raw []struct {
		IDStr       string `json:"id_str"`
		CreatedAt   string `json:"created_at"`
		FullText    string `json:"full_text"`
		Text        string `json:"text"`
		Lang        string `json:"lang"`
		FavoriteCount int  `json:"favorite_count"`
		RetweetCount  int  `json:"retweet_count"`
		// ReplyCount may not be present for v1.1; leave as 0 if absent
		User struct {
			IDStr string `json:"id_str"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]model.Tweet, 0, len(raw))
	for _, t := range raw {
		// Parse example: Mon Jan 2 15:04:05 -0700 2006
		ts, _ := time.Parse(time.RubyDate, t.CreatedAt)
		text := t.FullText
		if text == "" {
			text = t.Text
		}
		out = append(out, model.Tweet{
			ID:           t.IDStr,
			AuthorID:     t.User.IDStr,
			Text:         text,
			CreatedAt:    ts,
			Language:     t.Lang,
			LikeCount:    t.FavoriteCount,
			RetweetCount: t.RetweetCount,
		})
	}
	return out, nil
}

func (c *V1Client) oauth1Sign(req *http.Request, queryParams map[string]string) {
	oauth := map[string]string{
		"oauth_consumer_key":     c.ConsumerKey,
		"oauth_nonce":            c.nonceFn(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconv.FormatInt(c.nowFn().Unix(), 10),
		"oauth_token":            c.AccessToken,
		"oauth_version":          "1.0",
	}
	// Collect params
	all := map[string]string{}
	for k, v := range oauth {
		all[k] = v
	}
	for k, v := range queryParams {
		all[k] = v
	}
	// Parameter string
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	paramParts := make([]string, 0, len(keys))
	for _, k := range keys {
		paramParts = append(paramParts, rfc3986(k)+"="+rfc3986(all[k]))
	}
	paramStr := stringsJoinAmp(paramParts)
	baseURL := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
	base := "GET&" + rfc3986(baseURL) + "&" + rfc3986(paramStr)
	signingKey := rfc3986(c.ConsumerSecret) + "&" + rfc3986(c.AccessSecret)
	mac := hmac.New(sha1.New, []byte(signingKey))
	_, _ = mac.Write([]byte(base))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	oauth["oauth_signature"] = sig
	// Authorization header
	hdrKeys := make([]string, 0, len(oauth))
	for k := range oauth {
		hdrKeys = append(hdrKeys, k)
	}
	sort.Strings(hdrKeys)
	authParts := make([]string, 0, len(hdrKeys))
	for _, k := range hdrKeys {
		authParts = append(authParts, fmt.Sprintf("%s=\"%s\"", rfc3986(k), rfc3986(oauth[k])))
	}
	req.Header.Set("Authorization", "OAuth "+stringsJoinComma(authParts))
	req.Header.Set("Accept", "application/json")
}

func encodeQuery(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, urlEncode(k)+"="+urlEncode(m[k]))
	}
	return stringsJoinAmp(parts)
}

// RFC 3986 percent-encoding for OAuth
func rfc3986(s string) string { return strings.ReplaceAll(strings.ReplaceAll(urlEncode(s), "+", "%20"), "*", "%2A") }

func urlEncode(s string) string               { return url.QueryEscape(s) }
func stringsJoinAmp(parts []string) string    { return strings.Join(parts, "&") }
func stringsJoinComma(parts []string) string  { return strings.Join(parts, ", ") }
