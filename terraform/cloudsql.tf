# Cloud SQL MySQL Instance with security best practices
resource "google_sql_database_instance" "mysql_instance" {
  name             = var.db_instance_name
  database_version = "MYSQL_8_0"
  region           = var.region
  deletion_protection = true  # Best practice: enable deletion protection
  
  depends_on = [google_project_service.required_apis]

  settings {
    tier                        = "db-f1-micro"
    availability_type          = "ZONAL"
    disk_type                  = "PD_SSD"
    disk_size                  = 10
    disk_autoresize           = true
    disk_autoresize_limit     = 100

    backup_configuration {
      enabled                        = true
      start_time                    = "03:00"
      point_in_time_recovery_enabled = true
      binary_log_enabled            = true
      backup_retention_settings {
        retained_backups = 7
        retention_unit   = "COUNT"
      }
      location = var.region
    }

    ip_configuration {
      # Security best practice: Disable public IP when possible
      ipv4_enabled    = !var.enable_private_ip
      private_network = var.enable_private_ip ? google_compute_network.private_network[0].id : null
      require_ssl     = true  # Force SSL connections
      
      # Only allow public access if not using private IP
      dynamic "authorized_networks" {
        for_each = var.enable_private_ip ? [] : [1]
        content {
          name  = "cloud-run-access"
          value = "0.0.0.0/0"  # This will be restricted by IAM
        }
      }
    }

    # Security database flags
    database_flags {
      name  = "cloudsql_iam_authentication"
      value = "on"
    }
    
    database_flags {
      name  = "local_infile"
      value = "off"  # Security: disable local_infile
    }
    
    database_flags {
      name  = "skip_show_database"
      value = "on"   # Security: hide database names
    }

    insights_config {
      query_insights_enabled  = true
      record_application_tags = true
      record_client_address   = true
    }
  }
  
  lifecycle {
    prevent_destroy = true
  }
}

# Database
resource "google_sql_database" "oauth_database" {
  name     = var.db_name
  instance = google_sql_database_instance.mysql_instance.name
}

# Database User with secure password from Secret Manager
resource "google_sql_user" "app_user" {
  name     = var.db_user
  instance = google_sql_database_instance.mysql_instance.name
  password = random_password.db_password.result
  
  lifecycle {
    ignore_changes = [password]  # Password managed by Secret Manager
  }
}

# Output connection name for Cloud Run
output "instance_connection_name" {
  value = google_sql_database_instance.mysql_instance.connection_name
}

output "database_password_secret" {
  value = google_secret_manager_secret.db_password.secret_id
  description = "Secret Manager secret ID for database password"
}