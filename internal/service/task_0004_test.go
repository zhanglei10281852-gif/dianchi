package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
)

func TestRejectedManifestItemLeavesAggregateUntouched(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewManifestService()
	m := manifest.Manifest{ID: "manifest-4", Origin: "collection", Destination: "processor"}
	if err := svc.Create(ctx, p, m); err != nil {
		t.Fatal(err)
	}
	bad := manifest.Item{ID: "item-4", LotID: "lot-4", Weight: -10}
	if err := svc.AddItem(ctx, p, m.ID, bad); err == nil {
		t.Fatal("negative cargo item was accepted")
	}
	got, err := svc.Get(ctx, p, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("rejected cargo remained in manifest: %+v", got.Items)
	}
	good := manifest.Item{ID: "item-4", LotID: "lot-4", Weight: 10}
	if err := svc.AddItem(ctx, p, m.ID, good); err != nil {
		t.Fatalf("corrected cargo could not be added: %v", err)
	}
}
