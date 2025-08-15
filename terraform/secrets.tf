# Secret Manager for secure credential storage
resource "google_secret_manager_secret" "db_password" {
  secret_id = "${var.service_name}-db-password"
  
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

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db_password.result
}

# API Secret Key - to be created manually for security
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

# Client Secret Key - to be created manually for security  
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

# IAM permissions for Cloud Run to access secrets
resource "google_secret_manager_secret_iam_member" "cloudrun_secret_access" {
  for_each = {
    db_password        = google_secret_manager_secret.db_password.id
    api_secret_key     = google_secret_manager_secret.api_secret_key.id
    client_secret_key  = google_secret_manager_secret.client_secret_key.id
  }
  
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}