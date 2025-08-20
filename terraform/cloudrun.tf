# IAM binding moved to iam.tf for better organization

# Cloud Run Service with Secret Manager integration
resource "google_cloud_run_v2_service" "simple_relay" {
  name     = "${var.service_name}-${var.deploy_environment}"
  location = var.region

  depends_on = [
    google_firestore_database.oauth_database,
    google_cloud_run_v2_service.simple_billing  # Backend must wait for billing service
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
      image = "us-central1-docker.pkg.dev/${var.project_id}/${var.service_name}/simple-relay:${var.image_tag}"

      # Non-sensitive environment variables

      env {
        name  = "OFFICIAL_BASE_URL"
        value = var.official_base_url
      }

      env {
        name  = "DATABASE_TYPE"
        value = "firestore"
      }
      
      env {
        name  = "BILLING_SERVICE_URL"
        value = google_cloud_run_v2_service.simple_billing.uri
      }
      
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
      
      env {
        name  = "FIRESTORE_DATABASE_NAME"
        value = "${var.firestore_database_name}-${var.deploy_environment}"
      }

      # Secrets passed as environment variables
      env {
        name  = "API_SECRET_KEY"
        value = var.api_secret_key
      }

      env {
        name  = "ALLOWED_CLIENT_SECRET_KEY"
        value = var.client_secret_key
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

# Billing Cloud Run Service (Internal Only)
resource "google_cloud_run_v2_service" "simple_billing" {
  name     = "${var.billing_service_name}-${var.deploy_environment}"
  location = var.region

  depends_on = [
    google_firestore_database.oauth_database,
    google_project_iam_member.cloud_run_firestore_user
  ]

  template {
    # Security annotations
    annotations = {
      "run.googleapis.com/cpu-throttling"     = "false"
      "run.googleapis.com/execution-environment" = "gen2"
    }
    
    scaling {
      min_instance_count = 0
      max_instance_count = 5
    }

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/${var.service_name}/simple-billing:${var.image_tag}"

      env {
        name  = "BILLING_ENABLED"
        value = "true"
      }
      
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
      
      env {
        name  = "FIRESTORE_DATABASE_NAME"
        value = "${var.firestore_database_name}-${var.deploy_environment}"
      }

      ports {
        container_port = 8081
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
          port = 8081
        }
      }

      liveness_probe {
        initial_delay_seconds = 0
        timeout_seconds = 1
        period_seconds = 10
        failure_threshold = 3
        http_get {
          path = "/health"
          port = 8081
        }
      }
    }

    service_account = "${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  # Configure for all traffic to allow service-to-service communication
  ingress = "INGRESS_TRAFFIC_ALL"
}

# IAM policy for billing service - allow backend service to invoke billing service
resource "google_cloud_run_service_iam_member" "billing_internal_access" {
  service  = google_cloud_run_v2_service.simple_billing.name
  location = google_cloud_run_v2_service.simple_billing.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}


# Output the service URLs
output "service_url" {
  value = google_cloud_run_v2_service.simple_relay.uri
}

output "billing_service_url" {
  value = google_cloud_run_v2_service.simple_billing.uri
}

