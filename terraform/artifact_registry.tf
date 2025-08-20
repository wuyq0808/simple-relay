# Artifact Registry repository for Docker images
resource "google_artifact_registry_repository" "simple_relay" {
  location      = var.region
  repository_id = "simple-relay"
  description   = "Docker repository for Simple Relay application"
  format        = "DOCKER"
  
  depends_on = [google_project_service.required_apis]
}