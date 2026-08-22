package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/operator"
	"testing"
	"time"
)

func TestEcoOperatorCertificationSnapshot(t *testing.T) {
	s := NewOperatorService()
	p := auth.Principal{ID: "s", TenantID: "e", Role: auth.Supervisor}
	if e := s.Put(context.Background(), p, operator.Operator{ID: "w", Name: "W", Shift: operator.Day, Active: true, Certifications: map[operator.Certification]time.Time{operator.Hazmat: time.Now().Add(time.Hour)}}); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(context.Background(), p, "w")
	delete(got.Certifications, operator.Hazmat)
	fresh, _ := s.Get(context.Background(), p, "w")
	if _, ok := fresh.Certifications[operator.Hazmat]; !ok {
		t.Fatal("mutated")
	}
}
