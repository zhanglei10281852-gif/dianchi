package service

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
)

func newBatchFixture(t *testing.T) fixture {
	t.Helper()
	return newFixture(t)
}

func TestBatchImportConcurrentResultsComplete(t *testing.T) {
	f := newBatchFixture(t)
	defer f.close()
	const n = 200
	items := make([]BatchItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, BatchItem{Code: fmt.Sprintf("BATCH-%d", i), Chemistry: "LFP", Hazard: 1})
	}
	bs := BatchService{Lifecycle: f.svc, Workers: 16}
	out, err := bs.Import(f.ctx, f.operator, items, "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Created)+len(out.Failures) != n {
		t.Fatalf("accounted=%d want=%d (created=%d failures=%d)",
			len(out.Created)+len(out.Failures), n, len(out.Created), len(out.Failures))
	}
	if len(out.Failures) != 0 {
		t.Fatalf("unexpected failures=%d", len(out.Failures))
	}
	seen := map[string]bool{}
	for _, lot := range out.Created {
		if lot.State != battery.Received {
			t.Fatalf("lot state=%s", lot.State)
		}
		if seen[lot.Code] {
			t.Fatalf("duplicate created lot %s", lot.Code)
		}
		seen[lot.Code] = true
	}
}

func TestBatchImportRecordsMixedResultsConcurrently(t *testing.T) {
	f := newBatchFixture(t)
	defer f.close()
	// Half valid, half invalid chemistry so failures interleave with creates.
	items := make([]BatchItem, 0, 100)
	for i := 0; i < 100; i++ {
		chem := "LFP"
		if i%2 == 0 {
			chem = "BAD"
		}
		items = append(items, BatchItem{Code: fmt.Sprintf("MIX-%d", i), Chemistry: chem, Hazard: 1})
	}
	bs := BatchService{Lifecycle: f.svc, Workers: 16}
	out, err := bs.Import(f.ctx, f.operator, items, "mix-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Created)+len(out.Failures) != 100 {
		t.Fatalf("accounted=%d want=100 (created=%d failures=%d)",
			len(out.Created)+len(out.Failures), len(out.Created), len(out.Failures))
	}
	if len(out.Created) != 50 || len(out.Failures) != 50 {
		t.Fatalf("created=%d failures=%d want 50/50", len(out.Created), len(out.Failures))
	}
}

func TestBatchImportSummarySafeUnderConcurrency(t *testing.T) {
	f := newBatchFixture(t)
	defer f.close()
	const n = 100
	items := make([]BatchItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, BatchItem{Code: fmt.Sprintf("SUM-%d", i), Chemistry: "LFP", Hazard: 1})
	}
	bs := BatchService{Lifecycle: f.svc, Workers: 8}
	var wg sync.WaitGroup
	const runs = 5
	results := make([]string, runs)
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, _ := bs.Import(f.ctx, f.operator, items, fmt.Sprintf("sum-%d", i))
			results[i] = bs.Summary(out)
		}(i)
	}
	wg.Wait()
	for i, r := range results {
		if r == "" {
			t.Fatalf("run %d empty summary", i)
		}
	}
}
