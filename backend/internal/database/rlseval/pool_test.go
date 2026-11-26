//go:build integration && rlseval

package rlseval

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ADR-0010 evaluation requirement 2: connection pooling (pgxpool; no external pooler).

// TestPooledConnectionReuse: with one pooled connection, tenant A's transaction, a transaction with
// no context, and tenant B's transaction run on the same PostgreSQL backend process, and each sees
// only what its own context allows.
func TestPooledConnectionReuse(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	steps := []struct {
		tenant string
		want   []string
	}{
		{tenantA, rowsOf(tenantA)},
		{"", nil},
		{tenantB, rowsOf(tenantB)},
		{"", nil},
		{tenantA, rowsOf(tenantA)},
	}
	var firstPID int32
	for i, s := range steps {
		err := inTenant(ctx, pool, s.tenant, func(tx pgx.Tx) error {
			pid, err := backendPID(ctx, tx)
			if err != nil {
				return err
			}
			if i == 0 {
				firstPID = pid
			} else if pid != firstPID {
				return fmt.Errorf("step %d ran on backend %d, want the reused backend %d", i, pid, firstPID)
			}
			got, err := visible(ctx, tx, "rlseval.items")
			if err != nil {
				return err
			}
			if !equalRows(got, s.want) {
				return fmt.Errorf("step %d (tenant %q): visible = %v, want %v", i, s.tenant, got, s.want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestPoolConcurrentTenants: many goroutines share a small pool, each running transactions for a
// randomly chosen tenant (including tenants with no rows and no tenant at all). Every transaction
// must see exactly its own tenant's rows. Run with -race.
func TestPoolConcurrentTenants(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 4, nil)

	tenants := []string{tenantA, tenantB, tenantC, ""}
	const workers, iterations = 32, 40

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	pids := sync.Map{}
	for w := range workers {
		wg.Go(func() {
			for i := range iterations {
				tenant := tenants[(w+i)%len(tenants)]
				want := rowsOf(tenant)
				err := inTenant(ctx, pool, tenant, func(tx pgx.Tx) error {
					pid, err := backendPID(ctx, tx)
					if err != nil {
						return err
					}
					pids.Store(pid, true)
					got, err := visible(ctx, tx, "rlseval.items")
					if err != nil {
						return err
					}
					if !equalRows(got, want) {
						return fmt.Errorf("worker %d iteration %d tenant %q: visible = %v, want %v", w, i, tenant, got, want)
					}
					return nil
				})
				if err != nil {
					errs <- err
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	n := 0
	pids.Range(func(any, any) bool { n++; return true })
	t.Logf("%d transactions over %d pooled backend processes", workers*iterations, n)
	if n > 4 {
		t.Fatalf("used %d backends, want at most the pool size 4", n)
	}
}
