package tenancy

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
)

// MaxPageSize is the largest page List returns.
const MaxPageSize = 500

// Repository stores tenants in PostgreSQL through the sqlc-generated queries. It is safe for
// concurrent use.
//
// It connects as the application role (sms_app), which may read the tenants table, insert only a
// display name, and update only the status and updated_at (decision log §18).
type Repository struct {
	q *sqlcgen.Queries
}

// NewRepository returns a Repository that runs its queries through db: a pool (each method then
// runs in its own implicit transaction) or a transaction (pgx.Tx).
func NewRepository(db sqlcgen.DBTX) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

// Create registers a tenant with the given display name (normalized by NormalizeDisplayName). The
// database generates its UUIDv7 ID and timestamps; every tenant starts in StatusPending.
func (r *Repository) Create(ctx context.Context, displayName string) (Tenant, error) {
	name, err := NormalizeDisplayName(displayName)
	if err != nil {
		return Tenant{}, err
	}
	row, err := r.q.CreateTenant(ctx, name)
	if err != nil {
		return Tenant{}, fmt.Errorf("create tenant: %w", err)
	}
	return fromRow(row), nil
}

// Get returns the tenant with the given ID, or ErrNotFound.
func (r *Repository) Get(ctx context.Context, id ID) (Tenant, error) {
	if id.IsZero() {
		return Tenant{}, fmt.Errorf("%w: nil UUID", ErrInvalidID)
	}
	row, err := r.q.GetTenant(ctx, id.pg())
	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if err != nil {
		return Tenant{}, fmt.Errorf("get tenant %s: %w", id, err)
	}
	return fromRow(row), nil
}

// List returns up to pageSize tenants (1 to MaxPageSize) in ID order, starting after the tenant
// after. Pass the zero ID for the first page and the ID of the last tenant of the previous page
// for the next one; an empty result means there are no more tenants.
func (r *Repository) List(ctx context.Context, after ID, pageSize int) ([]Tenant, error) {
	if pageSize < 1 || pageSize > MaxPageSize {
		return nil, fmt.Errorf("%w: %d (allowed 1 to %d)", ErrInvalidPageSize, pageSize, MaxPageSize)
	}
	rows, err := r.q.ListTenants(ctx, sqlcgen.ListTenantsParams{
		AfterID:  after.pg(),
		PageSize: int32(pageSize), //nolint:gosec // G115: bounded to 1..MaxPageSize above.
	})
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	tenants := make([]Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = fromRow(row)
	}
	return tenants, nil
}

// Activate moves a pending tenant to active.
func (r *Repository) Activate(ctx context.Context, id ID) (Tenant, error) {
	return r.transition(ctx, id, StatusPending, StatusActive)
}

// Suspend moves an active tenant to suspended.
func (r *Repository) Suspend(ctx context.Context, id ID) (Tenant, error) {
	return r.transition(ctx, id, StatusActive, StatusSuspended)
}

// Reactivate moves a suspended tenant back to active.
func (r *Repository) Reactivate(ctx context.Context, id ID) (Tenant, error) {
	return r.transition(ctx, id, StatusSuspended, StatusActive)
}

// transition changes the status from -> to with one atomic, conditional UPDATE, so two concurrent
// transitions of the same tenant cannot both succeed. If no row changed, it reads the tenant to
// report ErrNotFound or ErrInvalidTransition (with the status found).
func (r *Repository) transition(ctx context.Context, id ID, from, to Status) (Tenant, error) {
	if id.IsZero() {
		return Tenant{}, fmt.Errorf("%w: nil UUID", ErrInvalidID)
	}
	row, err := r.q.TransitionTenantStatus(ctx, sqlcgen.TransitionTenantStatusParams{
		ToStatus:   string(to),
		ID:         id.pg(),
		FromStatus: string(from),
	})
	if err == nil {
		return fromRow(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, fmt.Errorf("change tenant %s status to %s: %w", id, to, err)
	}
	current, err := r.Get(ctx, id)
	if err != nil {
		return Tenant{}, err
	}
	return Tenant{}, fmt.Errorf("%w: tenant %s is %s; %s requires %s", ErrInvalidTransition, id, current.Status, to, from)
}

func fromRow(row sqlcgen.Tenant) Tenant {
	return Tenant{
		ID:          ID(row.ID.Bytes),
		DisplayName: row.DisplayName,
		Status:      Status(row.Status),
		CreatedAt:   row.CreatedAt.UTC(),
		UpdatedAt:   row.UpdatedAt.UTC(),
	}
}
