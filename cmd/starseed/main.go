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
    "starseed/internal/ingest"
    "starseed/internal/nn"
    "starseed/internal/store/sqlitevec"
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
    case "nn-train":
        cmdNNTrain()
    case "nn-infer":
        cmdNNInfer()
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
    fmt.Println("  nn-train    Train NN on 15-min features")
    fmt.Println("  nn-infer    Infer with NN on 15-min features")
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
    // Ingest recent tweets from followings
    tl, err := ingest.FromFollowing(ctx, client, follows, 5, *limit)
    if err != nil { fmt.Println("timeline error:", err) }
    fmt.Printf("Timeline ingested: %d tweets\n", len(tl))
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
    // Discovery by interests
    tweets, err := recommend.DiscoverTweetsByInterests(ctx, client, cfg, 50)
    if err == nil {
        fmt.Printf("Discovered %d interest-matched tweets\n", len(tweets))
    }
}

func cmdEngage() {
	fs := flag.NewFlagSet("engage", flag.ExitOnError)
	cfgPath := fs.String("config", "./starseed.yaml", "config path")
    seedFile := fs.String("seeds", "", "optional path to seed accounts file (one @handle per line)")
	_ = fs.Parse(os.Args[2:])
	cfg, err := config.Load(*cfgPath)
	if err != nil { fmt.Println("error:", err); os.Exit(1) }
    client := mustLoadClient(cfg)
    ctx := context.Background()
    now := time.Now().UTC()
    // If seed file is provided, expand discovery by those users' recent tweets
    var tweets []model.Tweet
    if *seedFile != "" {
        seeds, _ := readHandles(*seedFile)
        // Resolve handles to IDs
        for _, h := range seeds {
            u, err := client.GetUserByUsername(ctx, h)
            if err != nil { continue }
            ut, err := client.GetUserTweets(ctx, u.ID, 10)
            if err != nil { continue }
            tweets = append(tweets, ut...)
        }
    } else {
        // fallback: discover by interests
        found, _ := recommend.DiscoverTweetsByInterests(ctx, client, cfg, 50)
        tweets = append(tweets, found...)
    }
    sugs := suggest.HeuristicSuggest(tweets, now)
    // Respect quiet hours from config when scheduling
    qh := cfg.Engagement.QuietHours
    for i := range sugs {
        sugs[i].When = schedule.NextWindow(now, qh)
    }
    // Optionally upgrade with LLM
    for i := range sugs {
        upgraded, err := suggest.DraftWithLLM(ctx, cfg.LLM, sugs[i].Tweet.Text, sugs[i].Text)
        if err == nil && upgraded != "" { sugs[i].Text = upgraded }
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

func cmdNNTrain() {
    fs := flag.NewFlagSet("nn-train", flag.ExitOnError)
    cfgPath := fs.String("config", "./starseed.yaml", "config path")
    bin := fs.String("bin", "./starseed-nn/target/release/starseed-nn", "path to Rust NN binary")
    modelOut := fs.String("out", "./starseed_model.json", "output model path")
    hidden := fs.Int("hidden", 64, "hidden units")
    epochs := fs.Int("epochs", 10, "epochs")
    _ = fs.Parse(os.Args[2:])
    cfg, err := config.Load(*cfgPath)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    client := mustLoadClient(cfg)
    ctx := context.Background()
    me, err := client.GetUserByUsername(ctx, cfg.Account.Username)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    follows, err := client.GetFollowing(ctx, me.ID, 100)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    // Build samples over the last few hours from followings' tweets (proxy)
    timeline, _ := ingest.FromFollowing(ctx, client, follows, 5, 300)
    var samples []nn.FeatureVector
    // Persist features in vector DB for rolling and later training
    db, err := sqlitevec.Open(cfg.Storage.DBPath)
    if err != nil { fmt.Println("db error:", err); os.Exit(1) }
    defer db.Close()
    now := time.Now().UTC().Add(-6 * time.Hour)
    for w := 0; w < 24; w++ { // 6 hours in 15-min windows
        ws := now.Add(time.Duration(w) * 15 * time.Minute)
        fv, _ := nn.BuildFeaturesWithHistory(ctx, db, ws, timeline, nil)
        samples = append(samples, fv)
        _ = db.PutFeature(ctx, ws, fv.X, nil, map[string]any{"source":"train-window"})
    }
    if err := nn.Train(*bin, *modelOut, samples, *hidden, *epochs, 0.01); err != nil { fmt.Println("train error:", err); os.Exit(1) }
    fmt.Println("Model written to:", *modelOut)
}

func cmdNNInfer() {
    fs := flag.NewFlagSet("nn-infer", flag.ExitOnError)
    cfgPath := fs.String("config", "./starseed.yaml", "config path")
    bin := fs.String("bin", "./starseed-nn/target/release/starseed-nn", "path to Rust NN binary")
    modelPath := fs.String("model", "./starseed_model.json", "model path")
    _ = fs.Parse(os.Args[2:])
    cfg, err := config.Load(*cfgPath)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    client := mustLoadClient(cfg)
    ctx := context.Background()
    me, err := client.GetUserByUsername(ctx, cfg.Account.Username)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    follows, err := client.GetFollowing(ctx, me.ID, 100)
    if err != nil { fmt.Println("error:", err); os.Exit(1) }
    timeline, _ := ingest.FromFollowing(ctx, client, follows, 5, 100)
    ws := time.Now().UTC().Add(-15 * time.Minute)
    // open DB to leverage rolling history during inference feature build
    db, err := sqlitevec.Open(cfg.Storage.DBPath)
    if err != nil { fmt.Println("db error:", err); os.Exit(1) }
    defer db.Close()
    fv, _ := nn.BuildFeaturesWithHistory(ctx, db, ws, timeline, nil)
    preds, err := nn.Infer(*bin, *modelPath, []nn.FeatureVector{fv})
    if err != nil { fmt.Println("infer error:", err); os.Exit(1) }
    if len(preds) > 0 { fmt.Printf("pred next-window reply proxy: %.3f\n", preds[0][0]) }
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

func readHandles(path string) ([]string, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, err }
    lines := splitLines(string(b))
    var out []string
    for _, l := range lines {
        if l == "" { continue }
        if l[0] == '@' { l = l[1:] }
        out = append(out, l)
    }
    return out, nil
}

func splitLines(s string) []string {
    var out []string
    cur := ""
    for _, r := range s {
        if r == '\n' || r == '\r' { if cur != "" { out = append(out, cur); cur = "" }; continue }
        cur += string(r)
    }
    if cur != "" { out = append(out, cur) }
    return out
}

func scheduleNext(q []int) time.Time {
	return schedule.NextWindow(time.Now().UTC(), q)
}
