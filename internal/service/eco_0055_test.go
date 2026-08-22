package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"testing"
)

func TestEcoAuditWriteIsAtomicOnMarshalError(t *testing.T) {
	s := NewAuditLog()
	p := auth.Principal{ID: "o", TenantID: "e"}
	if e := s.Write(context.Background(), p, "lot", "inspect", "failed", "req", func() {}); e == nil {
		t.Fatal("accepted")
	}
	if got := s.Query(context.Background(), p, "lot"); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}
