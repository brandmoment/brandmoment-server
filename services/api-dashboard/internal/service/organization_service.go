package service

import (
	"services/api-dashboard/internal/models"
	"services/api-dashboard/internal/repo"
	"github.com/google/uuid"
)

type OrganizationService struct {
	repo *repo.OrganizationRepo
}

func NewOrganizationService(repo *repo.OrganizationRepo) *OrganizationService {
	return &OrganizationService{repo: repo}
}

func (s *OrganizationService) Create(org *models.Organization) error {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	return s.repo.Create(org)
}

func (s *OrganizationService) GetByID(id string) (*models.Organization, error) {
	return s.repo.GetByID(id)
}

func (s *OrganizationService) Update(org *models.Organization) error {
	return s.repo.Update(org)
}

func (s *OrganizationService) Delete(id string) error {
	return s.repo.Delete(id)
}