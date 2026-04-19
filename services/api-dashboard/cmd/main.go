package main

import (
	"database/sql"
	"log"
	"net/http"
	"services/api-dashboard/internal/api"
	"services/api-dashboard/internal/repo"
	"services/api-dashboard/internal/service"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dashboard?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	orgRepo := repo.NewOrganizationRepo(db)
	orgService := service.NewOrganizationService(orgRepo)
	orgHandler := api.NewOrganizationHandler(orgService)

	http.HandleFunc("/api/organizations", orgHandler.Create)
	http.HandleFunc("/api/organizations/", orgHandler.GetByID)
	http.HandleFunc("/api/organizations/", orgHandler.Update)
	http.HandleFunc("/api/organizations/", orgHandler.Delete)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}