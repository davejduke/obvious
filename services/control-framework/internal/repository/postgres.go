// Package repository provides the PostgreSQL implementation of the Repository interface.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository against a PostgreSQL database.
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// --- Framework operations ---

// ListFrameworks returns all frameworks visible to an org.
func (r *PostgresRepository) ListFrameworks(ctx context.Context, orgID uuid.UUID) ([]domain.Framework, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, org_id, name, version, authority, COALESCE(description,''), is_published, metadata, created_at, updated_at
		FROM control_frameworks
		WHERE org_id = $1
		ORDER BY name, version
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list frameworks: %w", err)
	}
	defer rows.Close()

	var frameworks []domain.Framework
	for rows.Next() {
		f, err := scanFramework(rows)
		if err != nil {
			return nil, err
		}
		frameworks = append(frameworks, *f)
	}
	return frameworks, rows.Err()
}

// GetFramework returns a single framework by ID.
func (r *PostgresRepository) GetFramework(ctx context.Context, id uuid.UUID) (*domain.Framework, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, org_id, name, version, authority, COALESCE(description,''), is_published, metadata, created_at, updated_at
		FROM control_frameworks WHERE id = $1
	`, id)
	f, err := scanFramework(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get framework: %w", err)
	}
	return f, nil
}

// CreateFramework inserts a new control framework.
func (r *PostgresRepository) CreateFramework(ctx context.Context, orgID uuid.UUID, req domain.CreateFrameworkRequest) (*domain.Framework, error) {
	meta, _ := json.Marshal(req.Metadata)
	var f domain.Framework
	err := r.db.QueryRow(ctx, `
		INSERT INTO control_frameworks (org_id, name, version, authority, description, is_published, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, org_id, name, version, authority, COALESCE(description,''), is_published, metadata, created_at, updated_at
	`, orgID, req.Name, req.Version, req.Authority, req.Description, req.IsPublished, meta,
	).Scan(&f.ID, &f.OrgID, &f.Name, &f.Version, &f.Authority, &f.Description, &f.IsPublished, &f.Metadata, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create framework: %w", err)
	}
	return &f, nil
}

// --- Control operations ---

// ListControls returns controls with filtering and pagination.
func (r *PostgresRepository) ListControls(ctx context.Context, orgID uuid.UUID, filter domain.ListControlsFilter) (*domain.PaginatedControls, error) {
	where := []string{"c.org_id = $1"}
	args := []interface{}{orgID}
	argN := 2

	if filter.FrameworkID != nil {
		where = append(where, fmt.Sprintf("c.framework_id = $%d", argN))
		args = append(args, *filter.FrameworkID)
		argN++
	}
	if filter.Domain != nil {
		where = append(where, fmt.Sprintf("c.domain = $%d", argN))
		args = append(args, *filter.Domain)
		argN++
	}
	if filter.ArticleRef != nil {
		where = append(where, fmt.Sprintf("c.article_ref = $%d", argN))
		args = append(args, *filter.ArticleRef)
		argN++
	}
	if filter.ParentID != nil {
		where = append(where, fmt.Sprintf("c.parent_id = $%d", argN))
		args = append(args, *filter.ParentID)
		argN++
	}
	if filter.IsActive != nil {
		where = append(where, fmt.Sprintf("c.is_active = $%d", argN))
		args = append(args, *filter.IsActive)
		argN++
	}
	if filter.Search != nil && *filter.Search != "" {
		where = append(where, fmt.Sprintf("(c.title ILIKE $%d OR c.description ILIKE $%d)", argN, argN))
		args = append(args, "%"+*filter.Search+"%")
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	// Count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM controls c %s`, whereClause), countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count controls: %w", err)
	}

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset

	query := fmt.Sprintf(`
		SELECT c.id, c.framework_id, c.org_id, c.parent_id, c.control_id, c.title,
			   COALESCE(c.description,''), COALESCE(c.objective,''), COALESCE(c.category,''),
			   COALESCE(c.domain,''), COALESCE(c.article_ref,''), COALESCE(c.implementation_notes,''),
			   c.test_procedures, c.evidence_requirements, c.risk_weight,
			   c.tags, c.is_active, c.created_at, c.updated_at
		FROM controls c
		%s
		ORDER BY c.control_id
		LIMIT $%d OFFSET $%d
	`, whereClause, argN, argN+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list controls: %w", err)
	}
	defer rows.Close()

	var controls []domain.Control
	for rows.Next() {
		c, err := scanControl(rows)
		if err != nil {
			return nil, err
		}
		controls = append(controls, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if controls == nil {
		controls = []domain.Control{}
	}

	return &domain.PaginatedControls{
		Controls: controls,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

// GetControl returns a single control by ID.
func (r *PostgresRepository) GetControl(ctx context.Context, id uuid.UUID) (*domain.Control, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, framework_id, org_id, parent_id, control_id, title,
			   COALESCE(description,''), COALESCE(objective,''), COALESCE(category,''),
			   COALESCE(domain,''), COALESCE(article_ref,''), COALESCE(implementation_notes,''),
			   test_procedures, evidence_requirements, risk_weight,
			   tags, is_active, created_at, updated_at
		FROM controls WHERE id = $1
	`, id)
	c, err := scanControl(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get control: %w", err)
	}
	return c, nil
}

// CreateControl inserts a new control.
func (r *PostgresRepository) CreateControl(ctx context.Context, orgID uuid.UUID, req domain.CreateControlRequest) (*domain.Control, error) {
	tpJSON, _ := json.Marshal(req.TestProcedures)
	erJSON, _ := json.Marshal(req.EvidenceRequirements)

	row := r.db.QueryRow(ctx, `
		INSERT INTO controls
			(framework_id, org_id, parent_id, control_id, title, description, objective,
			 category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, framework_id, org_id, parent_id, control_id, title,
			   COALESCE(description,''), COALESCE(objective,''), COALESCE(category,''),
			   COALESCE(domain,''), COALESCE(article_ref,''), COALESCE(implementation_notes,''),
			   test_procedures, evidence_requirements, risk_weight, tags, is_active, created_at, updated_at
	`,
		req.FrameworkID, orgID, req.ParentID, req.ControlID, req.Title, req.Description, req.Objective,
		req.Category, req.Domain, req.ArticleRef, tpJSON, erJSON, req.RiskWeight, req.Tags,
	)
	c, err := scanControl(row)
	if err != nil {
		return nil, fmt.Errorf("create control: %w", err)
	}
	return c, nil
}

// UpdateControl applies partial updates to a control.
func (r *PostgresRepository) UpdateControl(ctx context.Context, id uuid.UUID, req domain.UpdateControlRequest) (*domain.Control, error) {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argN))
		args = append(args, *req.Title)
		argN++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argN))
		args = append(args, *req.Description)
		argN++
	}
	if req.Objective != nil {
		sets = append(sets, fmt.Sprintf("objective = $%d", argN))
		args = append(args, *req.Objective)
		argN++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argN))
		args = append(args, *req.Category)
		argN++
	}
	if req.ImplementationNotes != nil {
		sets = append(sets, fmt.Sprintf("implementation_notes = $%d", argN))
		args = append(args, *req.ImplementationNotes)
		argN++
	}
	if req.TestProcedures != nil {
		b, _ := json.Marshal(req.TestProcedures)
		sets = append(sets, fmt.Sprintf("test_procedures = $%d", argN))
		args = append(args, b)
		argN++
	}
	if req.EvidenceRequirements != nil {
		b, _ := json.Marshal(req.EvidenceRequirements)
		sets = append(sets, fmt.Sprintf("evidence_requirements = $%d", argN))
		args = append(args, b)
		argN++
	}
	if req.RiskWeight != nil {
		sets = append(sets, fmt.Sprintf("risk_weight = $%d", argN))
		args = append(args, *req.RiskWeight)
		argN++
	}
	if req.Tags != nil {
		sets = append(sets, fmt.Sprintf("tags = $%d", argN))
		args = append(args, req.Tags)
		argN++
	}
	if req.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *req.IsActive)
		argN++
	}

	args = append(args, id)
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		UPDATE controls SET %s WHERE id = $%d
		RETURNING id, framework_id, org_id, parent_id, control_id, title,
			   COALESCE(description,''), COALESCE(objective,''), COALESCE(category,''),
			   COALESCE(domain,''), COALESCE(article_ref,''), COALESCE(implementation_notes,''),
			   test_procedures, evidence_requirements, risk_weight, tags, is_active, created_at, updated_at
	`, strings.Join(sets, ", "), argN), args...)

	c, err := scanControl(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("update control: %w", err)
	}
	return c, nil
}

// --- Mapping operations ---

// GetControlMappings returns all cross-framework mappings for a control.
func (r *PostgresRepository) GetControlMappings(ctx context.Context, controlID uuid.UUID) ([]domain.FrameworkMapping, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			fm.id, fm.source_framework_id, fm.target_framework_id,
			fm.source_control_id, fm.target_control_id,
			fm.mapping_type, fm.confidence, COALESCE(fm.notes,''), fm.created_at,
			tc.control_id AS target_control_ref,
			tc.title AS target_control_title,
			tf.name AS target_framework_name
		FROM framework_mappings fm
		JOIN controls tc ON tc.id = fm.target_control_id
		JOIN control_frameworks tf ON tf.id = fm.target_framework_id
		WHERE fm.source_control_id = $1
		ORDER BY tf.name, tc.control_id
	`, controlID)
	if err != nil {
		return nil, fmt.Errorf("get mappings: %w", err)
	}
	defer rows.Close()

	var mappings []domain.FrameworkMapping
	for rows.Next() {
		var m domain.FrameworkMapping
		err := rows.Scan(
			&m.ID, &m.SourceFrameworkID, &m.TargetFrameworkID,
			&m.SourceControlID, &m.TargetControlID,
			&m.MappingType, &m.Confidence, &m.Notes, &m.CreatedAt,
			&m.TargetControlRef, &m.TargetControlTitle, &m.TargetFrameworkName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		mappings = append(mappings, m)
	}
	if mappings == nil {
		mappings = []domain.FrameworkMapping{}
	}
	return mappings, rows.Err()
}

// --- Assessment operations ---

// GetOrCreateAssessment returns an existing assessment or creates a new one in not_started status.
func (r *PostgresRepository) GetOrCreateAssessment(ctx context.Context, engagementID, controlID, orgID uuid.UUID) (*domain.ControlAssessment, error) {
	// Upsert: if no assessment exists, create with not_started; otherwise return existing.
	var a domain.ControlAssessment
	err := r.db.QueryRow(ctx, `
		INSERT INTO control_assessments (engagement_id, control_id, org_id, status)
		VALUES ($1, $2, $3, 'not_started')
		ON CONFLICT (engagement_id, control_id) DO UPDATE
			SET updated_at = control_assessments.updated_at
		RETURNING id, engagement_id, control_id, org_id, status,
				  assessed_by_id, score, COALESCE(notes,''), evidence_ids,
				  COALESCE(transitioned_at, created_at), created_at, updated_at
	`, engagementID, controlID, orgID).Scan(
		&a.ID, &a.EngagementID, &a.ControlID, &a.OrgID, &a.Status,
		&a.AssessedByID, &a.Score, &a.Notes, &a.EvidenceIDs,
		&a.TransitionedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get or create assessment: %w", err)
	}
	return &a, nil
}

// UpdateAssessmentStatus applies a status transition (enforced by domain).
func (r *PostgresRepository) UpdateAssessmentStatus(ctx context.Context, assessmentID uuid.UUID, req domain.AssessControlRequest) (*domain.ControlAssessment, error) {
	var a domain.ControlAssessment
	err := r.db.QueryRow(ctx, `
		UPDATE control_assessments
		SET status = $1,
			score = $2,
			notes = $3,
			evidence_ids = $4,
			transitioned_at = NOW(),
			updated_at = NOW()
		WHERE id = $5
		RETURNING id, engagement_id, control_id, org_id, status,
				  assessed_by_id, score, COALESCE(notes,''), evidence_ids,
				  transitioned_at, created_at, updated_at
	`, req.Status, req.Score, req.Notes, req.EvidenceIDs, assessmentID).Scan(
		&a.ID, &a.EngagementID, &a.ControlID, &a.OrgID, &a.Status,
		&a.AssessedByID, &a.Score, &a.Notes, &a.EvidenceIDs,
		&a.TransitionedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("update assessment: %w", err)
	}
	return &a, nil
}

// GetAssessment returns the assessment for a given engagement + control pair.
func (r *PostgresRepository) GetAssessment(ctx context.Context, engagementID, controlID uuid.UUID) (*domain.ControlAssessment, error) {
	var a domain.ControlAssessment
	err := r.db.QueryRow(ctx, `
		SELECT id, engagement_id, control_id, org_id, status,
			   assessed_by_id, score, COALESCE(notes,''), evidence_ids,
			   COALESCE(transitioned_at, created_at), created_at, updated_at
		FROM control_assessments
		WHERE engagement_id = $1 AND control_id = $2
	`, engagementID, controlID).Scan(
		&a.ID, &a.EngagementID, &a.ControlID, &a.OrgID, &a.Status,
		&a.AssessedByID, &a.Score, &a.Notes, &a.EvidenceIDs,
		&a.TransitionedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get assessment: %w", err)
	}
	return &a, nil
}

// --- Scan helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanFramework(row scanner) (*domain.Framework, error) {
	var f domain.Framework
	var metaRaw []byte
	err := row.Scan(&f.ID, &f.OrgID, &f.Name, &f.Version, &f.Authority, &f.Description,
		&f.IsPublished, &metaRaw, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(metaRaw) > 0 {
		_ = json.Unmarshal(metaRaw, &f.Metadata)
	}
	return &f, nil
}

func scanControl(row scanner) (*domain.Control, error) {
	var c domain.Control
	var tpRaw, erRaw []byte
	err := row.Scan(
		&c.ID, &c.FrameworkID, &c.OrgID, &c.ParentID,
		&c.ControlID, &c.Title, &c.Description, &c.Objective, &c.Category,
		&c.Domain, &c.ArticleRef, &c.ImplementationNotes,
		&tpRaw, &erRaw, &c.RiskWeight, &c.Tags, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(tpRaw) > 0 {
		_ = json.Unmarshal(tpRaw, &c.TestProcedures)
	}
	if len(erRaw) > 0 {
		_ = json.Unmarshal(erRaw, &c.EvidenceRequirements)
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if c.TestProcedures == nil {
		c.TestProcedures = []domain.TestProcedure{}
	}
	if c.EvidenceRequirements == nil {
		c.EvidenceRequirements = []domain.EvidenceRequirement{}
	}
	return &c, nil
}

// MemoryRepository is an in-memory implementation for testing.
type MemoryRepository struct {
	frameworks  map[uuid.UUID]*domain.Framework
	controls    map[uuid.UUID]*domain.Control
	mappings    map[uuid.UUID][]domain.FrameworkMapping // keyed by sourceControlID
	assessments map[string]*domain.ControlAssessment   // keyed by "engagementID:controlID"
}

// NewMemoryRepository creates an empty in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		frameworks:  make(map[uuid.UUID]*domain.Framework),
		controls:    make(map[uuid.UUID]*domain.Control),
		mappings:    make(map[uuid.UUID][]domain.FrameworkMapping),
		assessments: make(map[string]*domain.ControlAssessment),
	}
}

func (m *MemoryRepository) ListFrameworks(ctx context.Context, orgID uuid.UUID) ([]domain.Framework, error) {
	var result []domain.Framework
	for _, f := range m.frameworks {
		if f.OrgID == orgID {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *MemoryRepository) GetFramework(ctx context.Context, id uuid.UUID) (*domain.Framework, error) {
	f, ok := m.frameworks[id]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (m *MemoryRepository) CreateFramework(ctx context.Context, orgID uuid.UUID, req domain.CreateFrameworkRequest) (*domain.Framework, error) {
	f := &domain.Framework{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        req.Name,
		Version:     req.Version,
		Authority:   req.Authority,
		Description: req.Description,
		IsPublished: req.IsPublished,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.frameworks[f.ID] = f
	return f, nil
}

func (m *MemoryRepository) ListControls(ctx context.Context, orgID uuid.UUID, filter domain.ListControlsFilter) (*domain.PaginatedControls, error) {
	var result []domain.Control
	for _, c := range m.controls {
		if c.OrgID != orgID {
			continue
		}
		if filter.FrameworkID != nil && c.FrameworkID != *filter.FrameworkID {
			continue
		}
		if filter.Domain != nil && c.Domain != *filter.Domain {
			continue
		}
		if filter.ArticleRef != nil && c.ArticleRef != *filter.ArticleRef {
			continue
		}
		if filter.IsActive != nil && c.IsActive != *filter.IsActive {
			continue
		}
		result = append(result, *c)
	}
	total := len(result)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = []domain.Control{}
	}
	return &domain.PaginatedControls{Controls: result, Total: total, Limit: limit, Offset: offset}, nil
}

func (m *MemoryRepository) GetControl(ctx context.Context, id uuid.UUID) (*domain.Control, error) {
	c, ok := m.controls[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *MemoryRepository) CreateControl(ctx context.Context, orgID uuid.UUID, req domain.CreateControlRequest) (*domain.Control, error) {
	c := &domain.Control{
		ID:                   uuid.New(),
		FrameworkID:          req.FrameworkID,
		OrgID:                orgID,
		ParentID:             req.ParentID,
		ControlID:            req.ControlID,
		Title:                req.Title,
		Description:          req.Description,
		Objective:            req.Objective,
		Category:             req.Category,
		Domain:               req.Domain,
		ArticleRef:           req.ArticleRef,
		ImplementationNotes:  req.ImplementationNotes,
		TestProcedures:       req.TestProcedures,
		EvidenceRequirements: req.EvidenceRequirements,
		RiskWeight:           req.RiskWeight,
		Tags:                 req.Tags,
		IsActive:             true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if c.TestProcedures == nil {
		c.TestProcedures = []domain.TestProcedure{}
	}
	if c.EvidenceRequirements == nil {
		c.EvidenceRequirements = []domain.EvidenceRequirement{}
	}
	m.controls[c.ID] = c
	return c, nil
}

func (m *MemoryRepository) UpdateControl(ctx context.Context, id uuid.UUID, req domain.UpdateControlRequest) (*domain.Control, error) {
	c, ok := m.controls[id]
	if !ok {
		return nil, nil
	}
	if req.Title != nil {
		c.Title = *req.Title
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.Objective != nil {
		c.Objective = *req.Objective
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if req.RiskWeight != nil {
		c.RiskWeight = *req.RiskWeight
	}
	if req.Tags != nil {
		c.Tags = req.Tags
	}
	c.UpdatedAt = time.Now()
	return c, nil
}

func (m *MemoryRepository) GetControlMappings(ctx context.Context, controlID uuid.UUID) ([]domain.FrameworkMapping, error) {
	maps := m.mappings[controlID]
	if maps == nil {
		return []domain.FrameworkMapping{}, nil
	}
	return maps, nil
}

func (m *MemoryRepository) GetOrCreateAssessment(ctx context.Context, engagementID, controlID, orgID uuid.UUID) (*domain.ControlAssessment, error) {
	key := fmt.Sprintf("%s:%s", engagementID, controlID)
	a, ok := m.assessments[key]
	if !ok {
		a = &domain.ControlAssessment{
			ID:             uuid.New(),
			EngagementID:   engagementID,
			ControlID:      controlID,
			OrgID:          orgID,
			Status:         domain.AssessmentStatusNotStarted,
			EvidenceIDs:    []uuid.UUID{},
			TransitionedAt: time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		m.assessments[key] = a
	}
	return a, nil
}

func (m *MemoryRepository) UpdateAssessmentStatus(ctx context.Context, assessmentID uuid.UUID, req domain.AssessControlRequest) (*domain.ControlAssessment, error) {
	key := fmt.Sprintf("%s:%s", req.EngagementID, req.EngagementID) // fallback
	for k, a := range m.assessments {
		if a.ID == assessmentID {
			key = k
			break
		}
	}
	a, ok := m.assessments[key]
	if !ok {
		return nil, nil
	}
	a.Status = req.Status
	a.Score = req.Score
	a.Notes = req.Notes
	a.TransitionedAt = time.Now()
	a.UpdatedAt = time.Now()
	return a, nil
}

func (m *MemoryRepository) GetAssessment(ctx context.Context, engagementID, controlID uuid.UUID) (*domain.ControlAssessment, error) {
	key := fmt.Sprintf("%s:%s", engagementID, controlID)
	a, ok := m.assessments[key]
	if !ok {
		return nil, nil
	}
	return a, nil
}

// AddMapping adds a mapping to the in-memory store (used for testing only).
func (m *MemoryRepository) AddMapping(sourceControlID uuid.UUID, mapping domain.FrameworkMapping) {
	m.mappings[sourceControlID] = append(m.mappings[sourceControlID], mapping)
}

