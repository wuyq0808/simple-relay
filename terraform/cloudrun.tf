# IAM binding for Cloud Run to access Firestore
resource "google_project_iam_member" "cloud_run_firestore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}

# Cloud Run Service with Secret Manager integration
resource "google_cloud_run_v2_service" "simple_relay" {
  name     = var.service_name
  location = var.region

  depends_on = [
    google_firestore_database.oauth_database,
    google_secret_manager_secret_iam_member.cloudrun_secret_access,
    google_secret_manager_secret_version.api_secret_key,
    google_secret_manager_secret_version.client_secret_key
  ]

  template {
    # Security annotations
    annotations = {
      "run.googleapis.com/cpu-throttling"     = "false"
      "run.googleapis.com/execution-environment" = "gen2"
    }
    
    # VPC Access for private IP
    dynamic "vpc_access" {
      for_each = var.enable_private_ip ? [1] : []
      content {
        connector = google_vpc_access_connector.connector[0].id
        egress    = "private-ranges-only"
      }
    }

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/${var.service_name}/${var.service_name}:${var.image_tag}"

      # Non-sensitive environment variables
      env {
        name  = "API_BASE_URL"
        value = var.api_base_url
      }

      env {
        name  = "OFFICIAL_BASE_URL"
        value = var.official_base_url
      }

      env {
        name  = "FIRESTORE_PROJECT_ID"
        value = var.project_id
      }

      env {
        name  = "DATABASE_TYPE"
        value = "firestore"
      }
      
      env {
        name  = "BILLING_ENABLED"
        value = "true"
      }
      
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }

      # Secrets from Secret Manager
      env {
        name = "API_SECRET_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.api_secret_key.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "ALLOWED_CLIENT_SECRET_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.client_secret_key.secret_id
            version = "latest"
          }
        }
      }


      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
        cpu_idle = true
        startup_cpu_boost = true
      }

      # Health checks
      startup_probe {
        initial_delay_seconds = 10
        timeout_seconds = 5
        period_seconds = 10
        failure_threshold = 30
        http_get {
          path = "/health"
          port = 8080
        }
      }

      liveness_probe {
        initial_delay_seconds = 0
        timeout_seconds = 1
        period_seconds = 10
        failure_threshold = 3
        http_get {
          path = "/health"
          port = 8080
        }
      }
    }

    service_account = "${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# IAM policy to allow unauthenticated access (consider restricting in production)
resource "google_cloud_run_service_iam_member" "public_access" {
  service  = google_cloud_run_v2_service.simple_relay.name
  location = google_cloud_run_v2_service.simple_relay.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Output the service URL
output "service_url" {
  value = google_cloud_run_v2_service.simple_relay.uri
}

output "secrets_to_populate" {
  value = {
    api_secret_key    = google_secret_manager_secret.api_secret_key.secret_id
    client_secret_key = google_secret_manager_secret.client_secret_key.secret_id
  }
  description = "Secret Manager secrets that need to be populated manually"
}