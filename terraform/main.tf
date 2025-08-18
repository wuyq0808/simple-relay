terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
  }
  required_version = ">= 1.0"
  
  # Best practice: Use Cloud Storage backend for state
  backend "gcs" {
    # Configure via backend config file or environment variables
    # bucket = "your-terraform-state-bucket"
    # prefix = "simple-relay"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Get current project information
data "google_project" "project" {}

# Variables
variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "simple-relay"
}



variable "official_base_url" {
  description = "Official base URL"
  type        = string
  default     = "https://console.anthropic.com"
}

variable "enable_private_ip" {
  description = "Enable private IP for Cloud SQL"
  type        = bool
  default     = false
}

variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

variable "api_secret_key" {
  description = "API Secret Key"
  type        = string
  sensitive   = true
}

variable "client_secret_key" {
  description = "Client Secret Key" 
  type        = string
  sensitive   = true
}

