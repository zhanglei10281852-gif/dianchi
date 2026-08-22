package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func TestConcurrentBatchImportPublishesResultsSafely(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	batch := BatchService{Lifecycle: New(store, clock.Fixed{Value: time.Now()}), Workers: 24}
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	items := make([]BatchItem, 200)
	for index := range items {
		items[index] = BatchItem{Code: fmt.Sprintf("LOT-28-%03d", index), Chemistry: "LFP", Hazard: 1}
	}
	result, err := batch.Import(ctx, p, items, "batch-28")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != len(items) || len(result.Failures) != 0 {
		t.Fatalf("batch outcomes lost under concurrency: created=%d failures=%d", len(result.Created), len(result.Failures))
	}
}
