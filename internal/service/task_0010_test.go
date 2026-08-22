package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
)

func TestRejectedQuarantineHoldDoesNotCreateRecord(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	svc := NewQuarantine()
	lot := battery.Lot{ID: "lot-10", State: battery.Received, ReceivedAt: time.Now().UTC()}
	if err := svc.Hold(ctx, p, lot, "not eligible"); err == nil {
		t.Fatal("non-quarantined lot was held")
	}
	if _, notes, ok := svc.Get(p, lot.ID); ok || len(notes) != 0 {
		t.Fatalf("rejected hold created quarantine record: ok=%v notes=%v", ok, notes)
	}
	lot.State = battery.Quarantined
	if err := svc.Hold(ctx, p, lot, "thermal inspection"); err != nil {
		t.Fatalf("eligible lot could not be held: %v", err)
	}
}
