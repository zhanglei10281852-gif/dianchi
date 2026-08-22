package service

import (
	"context"
	"testing"
	"time"
)

func TestCanceledRetentionPutDoesNotCreateRecord(t *testing.T) {
	base := context.Background()
	svc := NewRetentionStore()
	record := RetentionRecord{ID: "retention-16", Tenant: "plant-a", Kind: "audit", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Payload: []byte("original")}
	canceled, cancel := context.WithCancel(base)
	cancel()
	if err := svc.Put(canceled, record); err == nil {
		t.Fatal("canceled retention write succeeded")
	}
	if count := svc.Count(); count != 0 {
		t.Fatalf("canceled retention write created %d record", count)
	}
	record.ID = "retention-16-live"
	if err := svc.Put(base, record); err != nil {
		t.Fatal(err)
	}
	record.Payload[0] = 'X'
	got, ok := svc.Get(base, record.ID)
	if !ok || string(got.Payload) != "original" {
		t.Fatalf("stored retention payload was not isolated: %+v", got)
	}
}
