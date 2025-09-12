# Firestore Database
resource "google_firestore_database" "oauth_database" {
  project     = var.project_id
  name        = "${var.firestore_database_name}-${var.deploy_environment}"
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

# Firestore Index for hourly_aggregates collection - for cost limit range queries
resource "google_firestore_index" "hourly_aggregates_user_hour_asc" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "hourly_aggregates"

  fields {
    field_path = "user_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "hour"
    order      = "ASCENDING"
  }
}

# Firestore Index for hourly_aggregates collection - for descending hour queries
resource "google_firestore_index" "hourly_aggregates_user_hour_desc" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "hourly_aggregates"

  fields {
    field_path = "user_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "hour"
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

# Firestore Index for usage_records collection - by upstream account UUID and timestamp
resource "google_firestore_index" "usage_records_upstream_account_timestamp" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "usage_records"

  fields {
    field_path = "upstream_account_uuid"
    order      = "ASCENDING"
  }

  fields {
    field_path = "timestamp"
    order      = "DESCENDING"
  }
}

# Firestore Index for upstream_account_hourly_aggregates collection - supports range queries
# Document ID format: {upstream_account_uuid}_{hour} makes upstream_account_uuid effectively the partition key
resource "google_firestore_index" "upstream_account_hourly_aggregates_account_hour" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "upstream_account_hourly_aggregates"

  fields {
    field_path = "upstream_account_uuid"
    order      = "ASCENDING"
  }

  fields {
    field_path = "hour"
    order      = "ASCENDING"
  }
}

# Firestore Index for upstream_account_minute_aggregates collection - supports range queries
# Document ID format: {upstream_account_uuid}_{minute} makes upstream_account_uuid effectively the partition key
resource "google_firestore_index" "upstream_account_minute_aggregates_account_minute" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "upstream_account_minute_aggregates"

  fields {
    field_path = "upstream_account_uuid"
    order      = "ASCENDING"
  }

  fields {
    field_path = "minute"
    order      = "ASCENDING"
  }
}

# Firestore Index for upstream_account_minute_aggregates collection - descending minute order for recent data
resource "google_firestore_index" "upstream_account_minute_aggregates_account_minute_desc" {
  project    = var.project_id
  database   = google_firestore_database.oauth_database.name
  collection = "upstream_account_minute_aggregates"

  fields {
    field_path = "upstream_account_uuid"
    order      = "ASCENDING"
  }

  fields {
    field_path = "minute"
    order      = "DESCENDING"
  }
}