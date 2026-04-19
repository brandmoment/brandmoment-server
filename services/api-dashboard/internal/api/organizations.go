package api

import (
	"encoding/json"
	"net/http"
	"services/api-dashboard/internal/models"
	"services/api-dashboard/internal/service"
)

type OrganizationHandler struct {
	service *service.OrganizationService
}

func NewOrganizationHandler(service *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: service}
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var org models.Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if err := h.service.Create(&org); err != nil {
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	// For simplicity, this returns all organizations (in a real app you'd add pagination)
	// Implementation would call service.GetAll()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("[]"))
}

func (h *OrganizationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/organizations/"):]
	org, err := h.service.GetByID(id)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/organizations/"):]
	var org models.Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	org.ID, _ = uuid.Parse(id)
	if err := h.service.Update(&org); err != nil {
		http.Error(w, "Failed to update organization", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/organizations/"):]
	if err := h.service.Delete(id); err != nil {
		http.Error(w, "Failed to delete organization", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}