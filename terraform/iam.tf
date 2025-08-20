# IAM permissions for service accounts

# Grant the Terraform service account permission to manage Artifact Registry
# Note: This assumes you're using a service account for Terraform with the pattern:
# terraform@PROJECT_ID.iam.gserviceaccount.com
# If using GitHub Actions, this would be the service account specified in GCP_SA_KEY

# Get the current project data
data "google_project" "project" {
  project_id = var.project_id
}

# IAM binding for Cloud Run to access Firestore (moved from cloudrun.tf)
resource "google_project_iam_member" "cloud_run_firestore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}

# Grant Artifact Registry repository admin to the default compute service account
# This allows Cloud Build and Cloud Run to push/pull images
resource "google_project_iam_member" "compute_artifact_registry_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
}

# If you need to grant the Terraform service account permission to create repositories,
# uncomment and update the service account email below:
#
# resource "google_project_iam_member" "terraform_artifact_registry_admin" {
#   project = var.project_id
#   role    = "roles/artifactregistry.repoAdmin"
#   member  = "serviceAccount:terraform@${var.project_id}.iam.gserviceaccount.com"
# }