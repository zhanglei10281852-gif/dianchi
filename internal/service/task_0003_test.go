package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
)

func TestRejectedManifestMoveDoesNotPublishTargetState(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewManifestService()
	m := manifest.Manifest{ID: "manifest-3", Origin: "collection", Destination: "processor"}
	if err := svc.Create(ctx, p, m); err != nil {
		t.Fatal(err)
	}
	if err := svc.Move(ctx, p, m.ID, manifest.Received); err == nil {
		t.Fatal("prepared manifest was received directly")
	}
	got, err := svc.Get(ctx, p, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != manifest.Prepared || got.ReceivedAt != nil {
		t.Fatalf("rejected move leaked state: %+v", got)
	}
	if err := svc.Move(ctx, p, m.ID, manifest.Sealed); err != nil {
		t.Fatalf("legal sealing failed: %v", err)
	}
}
