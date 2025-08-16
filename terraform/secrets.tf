# Secret Manager for secure credential storage
# Note: No database password needed for Firestore (uses IAM)

# API Secret Key - create secret but populate externally
resource "google_secret_manager_secret" "api_secret_key" {
  secret_id = "${var.service_name}-api-secret-key"
  
  replication {
    auto {}
  }
  
  depends_on = [google_project_service.required_apis]
  
  labels = {
    app         = var.service_name
    environment = "production"
    managed_by  = "terraform"
  }
}

# Create initial version with placeholder (will be updated by workflow)
resource "google_secret_manager_secret_version" "api_secret_key" {
  secret      = google_secret_manager_secret.api_secret_key.id
  secret_data = "placeholder-will-be-updated"
  
  lifecycle {
    ignore_changes = [secret_data]
  }
}

# Client Secret Key - create secret but populate externally
resource "google_secret_manager_secret" "client_secret_key" {
  secret_id = "${var.service_name}-client-secret-key"
  
  replication {
    auto {}
  }
  
  depends_on = [google_project_service.required_apis]
  
  labels = {
    app         = var.service_name
    environment = "production"
    managed_by  = "terraform"
  }
}

# Create initial version with placeholder (will be updated by workflow)
resource "google_secret_manager_secret_version" "client_secret_key" {
  secret      = google_secret_manager_secret.client_secret_key.id
  secret_data = "placeholder-will-be-updated"
  
  lifecycle {
    ignore_changes = [secret_data]
  }
}

# IAM permissions for Cloud Run to access secrets
resource "google_secret_manager_secret_iam_member" "cloudrun_secret_access" {
  for_each = {
    api_secret_key     = google_secret_manager_secret.api_secret_key.id
    client_secret_key  = google_secret_manager_secret.client_secret_key.id
  }
  
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}