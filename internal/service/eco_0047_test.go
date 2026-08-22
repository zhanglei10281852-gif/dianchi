package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
	"testing"
)

func TestEcoManifestSealedRejectsItem(t *testing.T) {
	s := NewManifestService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.Create(context.Background(), p, manifest.Manifest{ID: "m", Origin: "a", Destination: "b"}); e != nil {
		t.Fatal(e)
	}
	if e := s.AddItem(context.Background(), p, "m", manifest.Item{ID: "i", LotID: "l", Weight: 1}); e != nil {
		t.Fatal(e)
	}
	if e := s.Move(context.Background(), p, "m", manifest.Sealed); e != nil {
		t.Fatal(e)
	}
	if e := s.AddItem(context.Background(), p, "m", manifest.Item{ID: "i2", LotID: "l2", Weight: 1}); e == nil {
		t.Fatal("sealed changed")
	}
}
