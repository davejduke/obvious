// Package repository provides data access for the identity service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/davejduke/obvious/services/identity/internal/models"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("repository: record not found")

// ErrDuplicate is returned when a unique constraint is violated.
var ErrDuplicate = errors.New("repository: duplicate record")

// Repository wraps the pgxpool and exposes typed query methods.
type Repository struct {
	db *pgxpool.Pool
}

// New creates a new Repository.
func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ---------- Organization ----------

// CreateOrganization inserts a new org and returns it.
func (r *Repository) CreateOrganization(ctx context.Context, name, slug, tier string) (*models.Organization, error) {
	const q = `
		INSERT INTO organizations (name, slug, tier)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, tier, industry, country_code, is_active, created_at, updated_at`
	var o models.Organization
	err := r.db.QueryRow(ctx, q, name, slug, tier).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Tier, &o.Industry, &o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, wrapErr("CreateOrganization", err)
	}
	return &o, nil
}

// GetOrganization returns an org by ID.
func (r *Repository) GetOrganization(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	const q = `SELECT id, name, slug, tier, industry, country_code, is_active, created_at, updated_at
	           FROM organizations WHERE id = $1`
	var o models.Organization
	err := r.db.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Tier, &o.Industry, &o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, wrapErr("GetOrganization", err)
}

// UpdateOrganization updates mutable org fields.
func (r *Repository) UpdateOrganization(ctx context.Context, id uuid.UUID, name, tier string) (*models.Organization, error) {
	const q = `UPDATE organizations SET name=$2, tier=$3, updated_at=NOW()
	           WHERE id=$1
	           RETURNING id, name, slug, tier, industry, country_code, is_active, created_at, updated_at`
	var o models.Organization
	err := r.db.QueryRow(ctx, q, id, name, tier).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Tier, &o.Industry, &o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, wrapErr("UpdateOrganization", err)
}

// ListOrganizations returns all active orgs (admin use only).
func (r *Repository) ListOrganizations(ctx context.Context) ([]*models.Organization, error) {
	const q = `SELECT id, name, slug, tier, industry, country_code, is_active, created_at, updated_at
	           FROM organizations WHERE is_active = true ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, wrapErr("ListOrganizations", err)
	}
	defer rows.Close()
	var orgs []*models.Organization
	for rows.Next() {
		var o models.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Tier, &o.Industry, &o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &o)
	}
	return orgs, rows.Err()
}

// ---------- Users ----------

// CreateUser inserts a new user.
func (r *Repository) CreateUser(ctx context.Context, orgID uuid.UUID, email, displayName, passwordHash string) (*models.User, error) {
	const q = `
		INSERT INTO users (org_id, email, display_name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, org_id, email, display_name, password_hash, is_active, last_login_at, created_at, updated_at`
	var u models.User
	err := r.db.QueryRow(ctx, q, orgID, email, displayName, passwordHash).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	return &u, wrapErr("CreateUser", err)
}

// GetUserByEmail finds a user within an org by email.
func (r *Repository) GetUserByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.User, error) {
	const q = `SELECT id, org_id, email, display_name, password_hash, is_active, last_login_at, created_at, updated_at
	           FROM users WHERE org_id=$1 AND email=$2`
	var u models.User
	err := r.db.QueryRow(ctx, q, orgID, email).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, wrapErr("GetUserByEmail", err)
}

// GetUserByID returns a user by ID, enforcing org isolation.
func (r *Repository) GetUserByID(ctx context.Context, orgID, userID uuid.UUID) (*models.User, error) {
	const q = `SELECT id, org_id, email, display_name, password_hash, is_active, last_login_at, created_at, updated_at
	           FROM users WHERE id=$1 AND org_id=$2`
	var u models.User
	err := r.db.QueryRow(ctx, q, userID, orgID).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, wrapErr("GetUserByID", err)
}

// ListUsers returns all active users within an org.
func (r *Repository) ListUsers(ctx context.Context, orgID uuid.UUID) ([]*models.User, error) {
	const q = `SELECT id, org_id, email, display_name, password_hash, is_active, last_login_at, created_at, updated_at
	           FROM users WHERE org_id=$1 AND is_active=true ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		return nil, wrapErr("ListUsers", err)
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// UpdateUser updates mutable user fields.
func (r *Repository) UpdateUser(ctx context.Context, orgID, userID uuid.UUID, displayName string) (*models.User, error) {
	const q = `UPDATE users SET display_name=$3, updated_at=NOW()
	           WHERE id=$1 AND org_id=$2
	           RETURNING id, org_id, email, display_name, password_hash, is_active, last_login_at, created_at, updated_at`
	var u models.User
	err := r.db.QueryRow(ctx, q, userID, orgID, displayName).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, wrapErr("UpdateUser", err)
}

// DeactivateUser soft-deletes a user.
func (r *Repository) DeactivateUser(ctx context.Context, orgID, userID uuid.UUID) error {
	const q = `UPDATE users SET is_active=false, updated_at=NOW() WHERE id=$1 AND org_id=$2`
	tag, err := r.db.Exec(ctx, q, userID, orgID)
	if err != nil {
		return wrapErr("DeactivateUser", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateLastLogin records the user's last login time.
func (r *Repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE users SET last_login_at=NOW(), updated_at=NOW() WHERE id=$1`
	_, err := r.db.Exec(ctx, q, userID)
	return wrapErr("UpdateLastLogin", err)
}

// ---------- Roles ----------

// GetRolesByUser returns all active roles for a user within an org.
func (r *Repository) GetRolesByUser(ctx context.Context, orgID, userID uuid.UUID) ([]*models.Role, error) {
	const q = `
		SELECT r.id, r.org_id, r.name, r.slug, r.description, r.is_system, r.created_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.org_id = $2
		  AND (ur.expires_at IS NULL OR ur.expires_at > NOW())`
	rows, err := r.db.Query(ctx, q, userID, orgID)
	if err != nil {
		return nil, wrapErr("GetRolesByUser", err)
	}
	defer rows.Close()
	var roles []*models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.OrgID, &role.Name, &role.Slug, &role.Description, &role.IsSystem, &role.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}
	return roles, rows.Err()
}

// GetSystemRoles returns all system roles for an org.
func (r *Repository) GetSystemRoles(ctx context.Context, orgID uuid.UUID) ([]*models.Role, error) {
	const q = `SELECT id, org_id, name, slug, description, is_system, created_at
	           FROM roles WHERE org_id=$1 ORDER BY name`
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		return nil, wrapErr("GetSystemRoles", err)
	}
	defer rows.Close()
	var roles []*models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.OrgID, &role.Name, &role.Slug, &role.Description, &role.IsSystem, &role.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}
	return roles, rows.Err()
}

// GetRoleBySlug fetches a role by slug within an org.
func (r *Repository) GetRoleBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*models.Role, error) {
	const q = `SELECT id, org_id, name, slug, description, is_system, created_at
	           FROM roles WHERE org_id=$1 AND slug=$2`
	var role models.Role
	err := r.db.QueryRow(ctx, q, orgID, slug).Scan(
		&role.ID, &role.OrgID, &role.Name, &role.Slug, &role.Description, &role.IsSystem, &role.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &role, wrapErr("GetRoleBySlug", err)
}

// AssignRole assigns a role to a user.
func (r *Repository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID, expiresAt *time.Time) error {
	const q = `
		INSERT INTO user_roles (user_id, role_id, granted_by, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, role_id) DO UPDATE SET granted_by=$3, expires_at=$4, granted_at=NOW()`
	_, err := r.db.Exec(ctx, q, userID, roleID, grantedBy, expiresAt)
	return wrapErr("AssignRole", err)
}

// EnsureSystemRoles creates the 7 persona roles for an org if they don't exist.
func (r *Repository) EnsureSystemRoles(ctx context.Context, orgID uuid.UUID) error {
	personas := []struct{ name, slug string }{
		{"Internal Auditor", "internal_auditor"},
		{"Chief Audit Executive", "cae"},
		{"Audit Committee", "audit_committee"},
		{"Auditee CISO", "auditee_ciso"},
		{"Cosourced Provider", "cosourced_provider"},
		{"PTWG Member", "ptwg_member"},
		{"Beta Tester", "beta_tester"},
	}
	for _, p := range personas {
		const q = `
			INSERT INTO roles (org_id, name, slug, is_system)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (org_id, slug) DO NOTHING`
		if _, err := r.db.Exec(ctx, q, orgID, p.name, p.slug); err != nil {
			return fmt.Errorf("EnsureSystemRoles %s: %w", p.slug, err)
		}
	}
	return nil
}

// GetFirstPersonaForUser returns the slug of the user's first active role.
func (r *Repository) GetFirstPersonaForUser(ctx context.Context, orgID, userID uuid.UUID) (string, error) {
	roles, err := r.GetRolesByUser(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	if len(roles) == 0 {
		return "beta_tester", nil
	}
	return roles[0].Slug, nil
}

// helper to normalise postgres errors
func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("repository.%s: %w", op, err)
}
