.PHONY: infra-up infra-down seed rill-ui

# Start all infra services (postgres, minio, rill, otel, jaeger)
infra-up:
	docker compose -f infra/docker/docker-compose.yml up -d

# Stop all infra services
infra-down:
	docker compose -f infra/docker/docker-compose.yml down

# Generate seed data and upload to MinIO
seed:
	cd infra/seed && go run .

# Open Rill UI
rill-ui:
	@echo "Rill UI: http://localhost:9009"
	@open http://localhost:9009 2>/dev/null || true

# Open Jaeger UI
jaeger-ui:
	@echo "Jaeger UI: http://localhost:16686"
	@open http://localhost:16686 2>/dev/null || true

# Open MinIO Console
minio-ui:
	@echo "MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
	@open http://localhost:9001 2>/dev/null || true
