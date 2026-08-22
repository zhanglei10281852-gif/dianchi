package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
	"testing"
)

func TestEcoPendingExcludesReceived(t *testing.T) {
	s := NewManifestService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.Create(context.Background(), p, manifest.Manifest{ID: "m", Origin: "a", Destination: "b"}); e != nil {
		t.Fatal(e)
	}
	if e := s.AddItem(context.Background(), p, "m", manifest.Item{ID: "i", LotID: "l", Weight: 1}); e != nil {
		t.Fatal(e)
	}
	for _, st := range []manifest.Status{manifest.Sealed, manifest.Dispatched, manifest.Received} {
		if e := s.Move(context.Background(), p, "m", st); e != nil {
			t.Fatal(e)
		}
	}
	if got := s.Pending("e"); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}
