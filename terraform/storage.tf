# Cloud Storage bucket for storing all API responses
resource "google_storage_bucket" "api_responses" {
  name     = "${var.project_id}-api-responses-${var.deploy_environment}"
  location = var.region

  # Prevent accidental deletion
  lifecycle {
    prevent_destroy = true
  }

  # Configure versioning
  versioning {
    enabled = true
  }

  # Configure lifecycle rules to manage storage costs
  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }

  # Uniform bucket-level access
  uniform_bucket_level_access = true

  # Public access prevention
  public_access_prevention = "enforced"
}

# Output the bucket name for use in application
output "api_responses_bucket" {
  description = "Name of the API responses storage bucket"
  value       = google_storage_bucket.api_responses.name
}