# Artifact Registry repository for Docker images
# This will create the repository if it doesn't exist
# If it already exists, import it with:
# terraform import google_artifact_registry_repository.simple_relay projects/PROJECT_ID/locations/REGION/repositories/simple-relay
resource "google_artifact_registry_repository" "simple_relay" {
  location      = var.region
  repository_id = "${var.service_name}-${var.environment}"
  description   = "Docker repository for Simple Relay application (${var.environment})"
  format        = "DOCKER"
  
  depends_on = [google_project_service.required_apis]
  
  lifecycle {
    # Prevent accidental deletion of the repository which would delete all images
    prevent_destroy = true
  }
}