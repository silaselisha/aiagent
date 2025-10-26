package ingest

import (
	"context"

	"starseed/internal/model"
	"starseed/internal/xclient"
)

// CollectAuthors maps author IDs to users using batched lookups.
func CollectAuthors(ctx context.Context, client xclient.XClient, tweets []model.Tweet) (map[string]model.User, error) {
	ids := make(map[string]struct{})
	for _, t := range tweets {
		if t.AuthorID != "" { ids[t.AuthorID] = struct{}{} }
	}
	// batch by 100
	arr := make([]string, 0, len(ids))
	for id := range ids { arr = append(arr, id) }
	out := make(map[string]model.User, len(arr))
	for i := 0; i < len(arr); i += 100 {
		end := i + 100
		if end > len(arr) { end = len(arr) }
		chunk := arr[i:end]
		users, err := client.GetUsersByIDs(ctx, chunk)
		if err != nil { return out, err }
		for _, u := range users { out[u.ID] = u }
	}
	return out, nil
}
