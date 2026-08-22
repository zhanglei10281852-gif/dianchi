package service

import (
	"context"
	"testing"
	"time"
)

func TestEcoRetentionPayloadSnapshot(t *testing.T) {
	s := NewRetentionStore()
	if e := s.Put(context.Background(), RetentionRecord{ID: "r", Tenant: "e", Kind: "m", Payload: []byte("evidence"), ExpiresAt: time.Now().Add(time.Hour)}); e != nil {
		t.Fatal(e)
	}
	got, ok := s.Get(context.Background(), "r")
	if !ok {
		t.Fatal("missing")
	}
	got.Payload[0] = 'X'
	fresh, _ := s.Get(context.Background(), "r")
	if string(fresh.Payload) != "evidence" {
		t.Fatalf("%q", fresh.Payload)
	}
}
