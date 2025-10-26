package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"starseed/internal/analytics"
	"starseed/internal/config"
    "starseed/internal/model"
	"starseed/internal/recommend"
    "starseed/internal/schedule"
	"starseed/internal/suggest"
	"starseed/internal/theme"
	"starseed/internal/xclient"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "init":
		cmdInit()
	case "analyze":
		cmdAnalyze()
	case "recommend":
		cmdRecommend()
	case "engage":
		cmdEngage()
	case "monitor":
		cmdMonitor()
	case "audit":
		cmdAudit()
	case "schedule":
		cmdSchedule()
	default:
		printHelp()
	}
}

func printHelp() {
	theme.PrintBanner()
	fmt.Println("Usage: starseed <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  init        Create a config file at ./starseed.yaml")
	fmt.Println("  analyze     Analyze timeline and followings")
	fmt.Println("  recommend   Recommend accounts and posts")
	fmt.Println("  engage      Suggest comments with timing")
	fmt.Println("  monitor     Show hourly engagement analytics")
	fmt.Println("  audit       Bot-likelihood and organic filters")
	fmt.Println("  schedule    Show next engagement window")
}

func mustLoadClient(cfg config.Config) *xclient.HTTPClient {
	if cfg.Credentials.BearerToken == "" {
		fmt.Println("warning: missing X_BEARER_TOKEN; API calls will fail")
	}
	return xclient.NewHTTPClient(cfg.Credentials.BearerToken)
}

func cmdInit() {
	out := flag.NewFlagSet("init", flag.ExitOnError)
	path := out.String("path", "./starseed.yaml", "path to write config")
	_ = out.Parse(os.Args[2:])
	cfg := config.Default()
	if err := config.Save(*path, cfg); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	abs, _ := filepath.Abs(*path)
	theme.PrintBanner()
	fmt.Println("Config written to:", abs)
}

func cmdAnalyze() {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	cfgPath := fs.String("config", "./starseed.yaml", "config path")
	limit := fs.Int("limit", 100, "items limit")
	_ = fs.Parse(os.Args[2:])
	cfg, err := config.Load(*cfgPath)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	client := mustLoadClient(cfg)
	ctx := context.Background()
	me, err := client.GetUserByUsername(ctx, cfg.Account.Username)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	follows, err := client.GetFollowing(ctx, me.ID, *limit)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	fmt.Printf("Following: %d users\n", len(follows))
	// Timeline placeholder: would fetch and score tweets here
	fmt.Println("Timeline analysis coming soon (API mapping)")
}

func cmdRecommend() {
	fs := flag.NewFlagSet("recommend", flag.ExitOnError)
	cfgPath := fs.String("config", "./starseed.yaml", "config path")
	_ = fs.Parse(os.Args[2:])
	cfg, err := config.Load(*cfgPath)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	client := mustLoadClient(cfg)
	ctx := context.Background()
	me, err := client.GetUserByUsername(ctx, cfg.Account.Username)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	follows, err := client.GetFollowing(ctx, me.ID, 200)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	recs := recommend.RankAccounts(follows, cfg.Interests.Keywords, cfg.Interests.Weights)
	for i := 0; i < len(recs) && i < 20; i++ {
		r := recs[i]
		fmt.Printf("@%s score=%.2f rel=%.2f bot=%.2f\n", r.User.Username, r.FinalScore, r.RelevanceScore, r.BotLikelihood)
	}
}

func cmdEngage() {
	fs := flag.NewFlagSet("engage", flag.ExitOnError)
	cfgPath := fs.String("config", "./starseed.yaml", "config path")
	_ = fs.Parse(os.Args[2:])
	cfg, err := config.Load(*cfgPath)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
	// Placeholder tweets; would come from timeline
	now := time.Now().UTC()
    tweets := []model.Tweet{
        {ID: "1", Text: "Kubernetes controllers are just control loops with reconciliation.", Language: "en"},
        {ID: "2", Text: "Golang generics removed most need for codegen in SDKs.", Language: "en"},
    }
    sugs := suggest.HeuristicSuggest(tweets, now)
    // Respect quiet hours from config when scheduling
    qh := cfg.Engagement.QuietHours
    for i := range sugs {
        sugs[i].When = schedule.NextWindow(now, qh)
    }
	for _, s := range sugs {
		fmt.Printf("when=%s why=%s\n%s\n---\n", s.When.Format(time.RFC3339), s.Why, s.Text)
	}
}

func cmdMonitor() {
	fs := flag.NewFlagSet("monitor", flag.ExitOnError)
	_ = fs.Parse(os.Args[2:])
	// Demo data
    events := []model.EngagementEvent{
        {Timestamp: time.Now().Add(-2 * time.Hour), Type: "reply"},
        {Timestamp: time.Now().Add(-2 * time.Hour), Type: "like"},
        {Timestamp: time.Now().Add(-1 * time.Hour), Type: "follow"},
    }
    b := analytics.HourlyEngagement(events)
	for _, k := range analytics.SortedBucketKeys(b) {
		fmt.Printf("%s -> %v\n", k.Format("15:00"), b[k])
	}
}

func cmdAudit() {
	fmt.Println("Audit will evaluate bot-likelihood and organic content filters (WIP).")
}

func cmdSchedule() {
	fs := flag.NewFlagSet("schedule", flag.ExitOnError)
	quiet := fs.String("quiet", "0,1,2,3,4,5", "quiet hours (UTC) comma-separated")
	_ = fs.Parse(os.Args[2:])
	qh := parseHours(*quiet)
	next := scheduleNext(qh)
	fmt.Println("Next window:", next.Format(time.RFC3339))
}

func parseHours(s string) []int {
	var out []int
	for _, p := range splitAndTrim(s) {
		var h int
		_, _ = fmt.Sscanf(p, "%d", &h)
		if h >= 0 && h <= 23 { out = append(out, h) }
	}
	return out
}

func splitAndTrim(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' { if cur != "" { out = append(out, cur); cur = "" }; continue }
		if r != ' ' { cur += string(r) }
	}
	if cur != "" { out = append(out, cur) }
	return out
}

func scheduleNext(q []int) time.Time {
	return schedule.NextWindow(time.Now().UTC(), q)
}
