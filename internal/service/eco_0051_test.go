package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"testing"
	"time"
)

func TestEcoQuarantineExpiryDropsNotes(t *testing.T) {
	q := NewQuarantine()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	l := battery.Lot{ID: "l", State: battery.Quarantined, ReceivedAt: time.Now().Add(-time.Hour)}
	if e := q.Hold(context.Background(), p, l, "old"); e != nil {
		t.Fatal(e)
	}
	if q.Expire(time.Now()) != 1 {
		t.Fatal("not expired")
	}
	l.ReceivedAt = time.Now()
	if e := q.Hold(context.Background(), p, l, "new"); e != nil {
		t.Fatal(e)
	}
	_, n, _ := q.Get(p, l.ID)
	if len(n) != 1 || n[0] != "new" {
		t.Fatalf("%v", n)
	}
}
