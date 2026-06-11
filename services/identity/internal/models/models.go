// Package models contains domain models specific to the identity service.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Persona represents one of the 7 AIAUDITOR RBAC personas.
type Persona string

const (
	PersonaInternalAuditor   Persona = "internal_auditor"
	PersonaCAE               Persona = "cae"
	PersonaAuditCommittee    Persona = "audit_committee"
	PersonaAuditeeCISO       Persona = "auditee_ciso"
	PersonaCosourcedProvider Persona = "cosourced_provider"
	PersonaPTWGMember        Persona = "ptwg_member"
	PersonaBetaTester        Persona = "beta_tester"
)

// AllPersonas enumerates all valid personas.
var AllPersonas = []Persona{
	PersonaInternalAuditor,
	PersonaCAE,
	PersonaAuditCommittee,
	PersonaAuditeeCISO,
	PersonaCosourcedProvider,
	PersonaPTWGMember,
	PersonaBetaTester,
}

// ValidPersona returns true if p is one of the 7 known personas.
func ValidPersona(p Persona) bool {
	for _, v := range AllPersonas {
		if v == p {
			return true
		}
	}
	return false
}

// Organization represents a tenant organisation (maps to organizations table).
type Organization struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Slug        string    `db:"slug"        json:"slug"`
	Tier        string    `db:"tier"        json:"tier"`
	Industry    *string   `db:"industry"    json:"industry,omitempty"`
	CountryCode *string   `db:"country_code" json:"country_code,omitempty"`
	IsActive    bool      `db:"is_active"   json:"is_active"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}

// User represents a platform user (maps to users table).
type User struct {
	ID           uuid.UUID  `db:"id"           json:"id"`
	OrgID        uuid.UUID  `db:"org_id"       json:"org_id"`
	Email        string     `db:"email"        json:"email"`
	DisplayName  string     `db:"display_name" json:"display_name"`
	PasswordHash *string    `db:"password_hash" json:"-"`
	IsActive     bool       `db:"is_active"    json:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"   json:"updated_at"`
}

// Role represents a role in the system.
type Role struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	OrgID       uuid.UUID `db:"org_id"      json:"org_id"`
	Name        string    `db:"name"        json:"name"`
	Slug        string    `db:"slug"        json:"slug"`
	Description *string   `db:"description" json:"description,omitempty"`
	IsSystem    bool      `db:"is_system"   json:"is_system"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// UserRole links a user to a role.
type UserRole struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	UserID    uuid.UUID  `db:"user_id"    json:"user_id"`
	RoleID    uuid.UUID  `db:"role_id"    json:"role_id"`
	GrantedBy *uuid.UUID `db:"granted_by" json:"granted_by,omitempty"`
	GrantedAt time.Time  `db:"granted_at" json:"granted_at"`
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at,omitempty"`
}

// RefreshToken stored in Redis. Key = "refresh:<token_id>".
type RefreshToken struct {
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Claims is the standard set of JWT claims issued by identity service.
type Claims struct {
	UserID      string  `json:"sub"`
	OrgID       string  `json:"org_id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"name"`
	Persona     Persona `json:"persona"`
	TokenID     string  `json:"jti"`
}
