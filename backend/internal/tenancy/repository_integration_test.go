//go:build integration

package tenancy_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// The repository is exercised exactly as the API will use it: through the application role
// (sms_app), checked by database.RequireApplicationRole, on a database migrated by the migration
// role. There are no HTTP endpoints.

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func newRepository(t *testing.T) (*tenancy.Repository, *database.DB) {
	t.Helper()
	roles := dbtest.NewMigratedDatabase(t)
	db, err := database.Open(context.Background(), roles.App, "sms-tenancy-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(db.Close)
	return tenancy.NewRepository(db.Pool()), db
}

func TestIntegrationCreateGeneratesUUIDv7(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, db := newRepository(t)

	before := time.Now()
	var previous tenancy.ID
	for i := range 20 {
		tenant, err := repo.Create(ctx, "  Example Tenant  ")
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		if tenant.ID.Version() != 7 {
			t.Errorf("tenant ID %s has version %d, want 7", tenant.ID, tenant.ID.Version())
		}
		if tenant.ID[8]&0xc0 != 0x80 {
			t.Errorf("tenant ID %s does not have the RFC 9562 variant", tenant.ID)
		}
		if err := database.ValidateTenantID(tenant.ID); err != nil {
			t.Errorf("tenant ID %s rejected by the tenant-context helper: %v", tenant.ID, err)
		}
		// PostgreSQL agrees: version 7, and the embedded timestamp is the creation time.
		var version int
		var embedded time.Time
		if err := db.Pool().QueryRow(ctx, `SELECT uuid_extract_version($1::uuid), uuid_extract_timestamp($1::uuid)`,
			tenant.ID.String()).Scan(&version, &embedded); err != nil {
			t.Fatalf("inspect UUID: %v", err)
		}
		if version != 7 {
			t.Errorf("uuid_extract_version(%s) = %d, want 7", tenant.ID, version)
		}
		if d := embedded.Sub(tenant.CreatedAt); d < -time.Second || d > time.Second {
			t.Errorf("UUID timestamp %s differs from created_at %s by %s", embedded, tenant.CreatedAt, d)
		}
		// Time-ordered: each new ID sorts after the previous one.
		if i > 0 && string(tenant.ID[:]) <= string(previous[:]) {
			t.Errorf("tenant ID %s does not sort after the previous %s", tenant.ID, previous)
		}
		previous = tenant.ID

		if tenant.DisplayName != "Example Tenant" || tenant.Status != tenancy.StatusPending {
			t.Errorf("Create() = %+v, want display name trimmed and status pending", tenant)
		}
		if tenant.CreatedAt.Location() != time.UTC || !tenant.UpdatedAt.Equal(tenant.CreatedAt) ||
			tenant.CreatedAt.Before(before.Add(-time.Minute)) {
			t.Errorf("Create() timestamps = %s, %s", tenant.CreatedAt, tenant.UpdatedAt)
		}
	}
}

func TestIntegrationDisplayNamesNeedNotBeUnique(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, _ := newRepository(t)
	a, errA := repo.Create(ctx, "Same Name")
	b, errB := repo.Create(ctx, "Same Name")
	if errA != nil || errB != nil || a.ID == b.ID {
		t.Fatalf("two tenants with the same name: %v, %v (IDs %s, %s)", errA, errB, a.ID, b.ID)
	}
}

func TestIntegrationCreateRejectsInvalidDisplayNames(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, db := newRepository(t)
	for _, name := range []string{"", "   ", "two\nlines"} {
		if _, err := repo.Create(ctx, name); !errors.Is(err, tenancy.ErrInvalidDisplayName) {
			t.Errorf("Create(%q) error = %v, want ErrInvalidDisplayName", name, err)
		}
	}
	// The database enforces the same rule if the repository is bypassed.
	for _, name := range []string{" untrimmed", "", "two\nlines"} {
		_, err := db.Pool().Exec(ctx, `INSERT INTO tenants (display_name) VALUES ($1)`, name)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != "ck_tenants__display_name" {
			t.Errorf("INSERT display_name %q: error = %v, want check violation of ck_tenants__display_name", name, err)
		}
	}
}

func TestIntegrationGet(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, _ := newRepository(t)
	created, err := repo.Create(ctx, "Get Me")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil || got != created {
		t.Errorf("Get() = %+v, %v; want %+v", got, err, created)
	}
	missing, err := tenancy.ParseID("019abc12-3456-7def-8123-456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, missing); !errors.Is(err, tenancy.ErrNotFound) {
		t.Errorf("Get(unknown) error = %v, want ErrNotFound", err)
	}
	if _, err := repo.Get(ctx, tenancy.ID{}); !errors.Is(err, tenancy.ErrInvalidID) {
		t.Errorf("Get(nil UUID) error = %v, want ErrInvalidID", err)
	}
}

func TestIntegrationList(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, _ := newRepository(t)

	empty, err := repo.List(ctx, tenancy.ID{}, 10)
	if err != nil || len(empty) != 0 || empty == nil {
		t.Fatalf("List() on an empty registry = %v, %v; want an empty, non-nil slice", empty, err)
	}

	var created []tenancy.ID
	for _, name := range []string{"T1", "T2", "T3", "T4", "T5"} {
		tenant, err := repo.Create(ctx, name)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		created = append(created, tenant.ID)
	}

	var listed []tenancy.ID
	var sizes []int
	after := tenancy.ID{}
	for {
		page, err := repo.List(ctx, after, 2)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if len(page) == 0 {
			break
		}
		sizes = append(sizes, len(page))
		for _, tenant := range page {
			listed = append(listed, tenant.ID)
		}
		after = page[len(page)-1].ID
	}
	if len(sizes) != 3 || sizes[0] != 2 || sizes[1] != 2 || sizes[2] != 1 {
		t.Errorf("page sizes = %v, want [2 2 1]", sizes)
	}
	if len(listed) != len(created) {
		t.Fatalf("listed %d tenants, want %d", len(listed), len(created))
	}
	for i := range created { // UUIDv7 order is creation order
		if listed[i] != created[i] {
			t.Errorf("listed[%d] = %s, want %s", i, listed[i], created[i])
		}
	}

	for _, size := range []int{0, -1, tenancy.MaxPageSize + 1} {
		if _, err := repo.List(ctx, tenancy.ID{}, size); !errors.Is(err, tenancy.ErrInvalidPageSize) {
			t.Errorf("List(page size %d) error = %v, want ErrInvalidPageSize", size, err)
		}
	}
}

func TestIntegrationStatusLifecycle(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, _ := newRepository(t)
	tenant, err := repo.Create(ctx, "Lifecycle")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	id := tenant.ID

	type op struct {
		name string
		fn   func(context.Context, tenancy.ID) (tenancy.Tenant, error)
	}
	activate := op{"Activate", repo.Activate}
	suspend := op{"Suspend", repo.Suspend}
	reactivate := op{"Reactivate", repo.Reactivate}

	steps := []struct {
		op      op
		want    tenancy.Status // status after the step
		wantErr error
	}{
		{suspend, tenancy.StatusPending, tenancy.ErrInvalidTransition},    // pending cannot be suspended
		{reactivate, tenancy.StatusPending, tenancy.ErrInvalidTransition}, // nor reactivated
		{activate, tenancy.StatusActive, nil},
		{activate, tenancy.StatusActive, tenancy.ErrInvalidTransition},
		{reactivate, tenancy.StatusActive, tenancy.ErrInvalidTransition},
		{suspend, tenancy.StatusSuspended, nil},
		{suspend, tenancy.StatusSuspended, tenancy.ErrInvalidTransition},
		{activate, tenancy.StatusSuspended, tenancy.ErrInvalidTransition}, // suspended is reactivated, not activated
		{reactivate, tenancy.StatusActive, nil},
		{suspend, tenancy.StatusSuspended, nil},
		{reactivate, tenancy.StatusActive, nil},
	}
	last := tenant
	for i, s := range steps {
		got, err := s.op.fn(ctx, id)
		if s.wantErr != nil {
			if !errors.Is(err, s.wantErr) {
				t.Fatalf("step %d: %s() error = %v, want %v", i+1, s.op.name, err, s.wantErr)
			}
		} else {
			if err != nil {
				t.Fatalf("step %d: %s() error: %v", i+1, s.op.name, err)
			}
			if got.Status != s.want || got.UpdatedAt.Before(last.UpdatedAt) || !got.CreatedAt.Equal(tenant.CreatedAt) ||
				got.DisplayName != tenant.DisplayName {
				t.Errorf("step %d: %s() = %+v", i+1, s.op.name, got)
			}
			last = got
		}
		stored, err := repo.Get(ctx, id)
		if err != nil || stored.Status != s.want {
			t.Fatalf("step %d: after %s(), stored status = %s, %v; want %s", i+1, s.op.name, stored.Status, err, s.want)
		}
	}

	missing, err := tenancy.ParseID("019abc12-3456-7def-8123-456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []op{activate, suspend, reactivate} {
		if _, err := o.fn(ctx, missing); !errors.Is(err, tenancy.ErrNotFound) {
			t.Errorf("%s(unknown) error = %v, want ErrNotFound", o.name, err)
		}
		if _, err := o.fn(ctx, tenancy.ID{}); !errors.Is(err, tenancy.ErrInvalidID) {
			t.Errorf("%s(nil UUID) error = %v, want ErrInvalidID", o.name, err)
		}
	}
}

// TestIntegrationConcurrentTransitionsSucceedOnce: many concurrent activations of one pending
// tenant; exactly one succeeds and the others report an invalid transition.
func TestIntegrationConcurrentTransitionsSucceedOnce(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	repo, _ := newRepository(t)
	tenant, err := repo.Create(ctx, "Race")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Go(func() {
			_, err := repo.Activate(ctx, tenant.ID)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	var succeeded, rejected int
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, tenancy.ErrInvalidTransition):
			rejected++
		default:
			t.Errorf("Activate() unexpected error: %v", err)
		}
	}
	if succeeded != 1 || rejected != workers-1 {
		t.Errorf("concurrent activations: %d succeeded, %d rejected; want 1 and %d", succeeded, rejected, workers-1)
	}
}
