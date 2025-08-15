# Enable required APIs - Best practice: explicit API enablement
resource "google_project_service" "required_apis" {
  for_each = toset([
    "secretmanager.googleapis.com",
    "sqladmin.googleapis.com", 
    "run.googleapis.com",
    "compute.googleapis.com",
    "servicenetworking.googleapis.com",
    "cloudbuild.googleapis.com"
  ])
  
  service            = each.value
  disable_on_destroy = false
  
  # Prevent destruction of dependent resources
  lifecycle {
    prevent_destroy = true
  }
}