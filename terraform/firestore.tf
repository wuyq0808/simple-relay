# Firestore Database
resource "google_firestore_database" "oauth_database" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
  
  depends_on = [google_project_service.required_apis]
}

# Firestore Index for usage_records collection
resource "google_firestore_index" "usage_records_user_timestamp" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "usage_records"

  fields {
    field_path = "user_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "timestamp"
    order      = "DESCENDING"
  }
}

# Firestore Index for daily_aggregates collection
resource "google_firestore_index" "daily_aggregates_user_date" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "daily_aggregates"

  fields {
    field_path = "user_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "date"
    order      = "DESCENDING"
  }
}

# Firestore Index for usage_records collection - by model
resource "google_firestore_index" "usage_records_model_timestamp" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "usage_records"

  fields {
    field_path = "model"
    order      = "ASCENDING"
  }

  fields {
    field_path = "timestamp"
    order      = "DESCENDING"
  }
}