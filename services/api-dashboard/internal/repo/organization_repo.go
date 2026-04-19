package repo

import (
	"services/api-dashboard/internal/models"
	"database/sql"
)

type OrganizationRepo struct {
	db *sql.DB
}

func NewOrganizationRepo(db *sql.DB) *OrganizationRepo {
	return &OrganizationRepo{db: db}
}

func (r *OrganizationRepo) Create(org *models.Organization) error {
	_, err := r.db.Exec(`
		INSERT INTO organizations (id, name, slug, org_type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, org.ID, org.Name, org.Slug, org.OrgType, org.IsActive, org.CreatedAt, org.UpdatedAt)
	return err
}

func (r *OrganizationRepo) GetByID(id string) (*models.Organization, error) {
	var org models.Organization
	err := r.db.QueryRow(`
		SELECT id, name, slug, org_type, is_active, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.OrgType, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepo) Update(org *models.Organization) error {
	_, err := r.db.Exec(`
		UPDATE organizations
		SET name = $1, slug = $2, org_type = $3, is_active = $4, updated_at = $5
		WHERE id = $6
	`, org.Name, org.Slug, org.OrgType, org.IsActive, org.UpdatedAt, org.ID)
	return err
}

func (r *OrganizationRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	return err
}