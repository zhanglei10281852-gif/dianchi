package service

import (
	"context"
	"errors"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
	"testing"
	"time"
)

type ecoFailPublisher struct{}

func (ecoFailPublisher) Publish(context.Context, string, string) error {
	return errors.New("broker down")
}
func TestEcoOutboxFailureIsRecorded(t *testing.T) {
	st, e := sqlite.Open(context.Background(), "file:eco43?mode=memory&cache=shared")
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, e = st.DB.Exec("INSERT INTO outbox(id,tenant_id,aggregate_id,event_type,payload,state,next_attempt_at,created_at) VALUES('e43','t','lot','event','{}','pending',?,?)", now, now); e != nil {
		t.Fatal(e)
	}
	w := OutboxWorker{Store: st, Publisher: ecoFailPublisher{}, Clock: func() time.Time { return time.Now().UTC() }}
	if e = w.RunOnce(context.Background()); e == nil {
		t.Fatal("expected error")
	}
	var state string
	if e = st.DB.QueryRow("SELECT state FROM outbox WHERE id='e43'").Scan(&state); e != nil {
		t.Fatal(e)
	}
	if state == "published" {
		t.Fatal("published")
	}
}
