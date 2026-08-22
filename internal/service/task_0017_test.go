package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
)

func TestAuditMarshalFailureLeavesNoEvent(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "auditor", TenantID: "plant-a", Role: auth.Supervisor}
	svc := NewAuditLog()
	if err := svc.Write(ctx, p, "lot-17", "archive", "failed", "request-17", make(chan int)); err == nil {
		t.Fatal("unsupported audit payload was accepted")
	}
	if rows := svc.Query(ctx, p, "lot-17"); len(rows) != 0 {
		t.Fatalf("marshal failure left phantom audit event: %+v", rows)
	}
	if err := svc.Write(ctx, p, "lot-17", "archive", "success", "request-18", map[string]string{"state": "kept"}); err != nil {
		t.Fatalf("encodable audit payload failed: %v", err)
	}
}
