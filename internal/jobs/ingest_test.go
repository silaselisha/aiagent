package jobs

import (
	"context"
	"testing"
	"time"

	"starseed/internal/config"
	"starseed/internal/store/sqlitevec"
)

type fakeClient struct{}

func (f fakeClient) GetUserByUsername(ctx context.Context, username string) (struct{ ID, Username string }, error) {
	return struct{ ID, Username string }{ID: "me", Username: username}, nil
}

func TestRunIngestionOnce_Noop(t *testing.T) {
	db, err := sqlitevec.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	ctx := context.Background()
	cfg := config.Default()
	cfg.Account.Username = "me"
	// We can't fully run without a real xclient; this is a harness placeholder
	_ = ctx; _ = cfg
	_ = time.Second
}
