package service

import (
	"context"
	"testing"
)

func TestEcoReconciliationFailedSaveDoesNotPublishCache(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	r, e := svc.CreateReport(context.Background(), op, "eco-0033")
	if e != nil {
		t.Fatal(e)
	}
	store.fail = true
	if _, e = svc.AppendMeasurement(context.Background(), op, r.ID, recMeasurement("lot", 91)); e == nil {
		t.Fatal("expected failure")
	}
	store.fail = false
	got, e := svc.load(context.Background(), op, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Measurements) != 0 {
		t.Fatalf("leaked: %+v", got.Measurements)
	}
}
