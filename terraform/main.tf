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
    # prefix will be set via -backend-config during terraform init
    # Example: terraform init -backend-config="prefix=simple-relay-production"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}


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

variable "billing_service_name" {
  description = "Name of the billing Cloud Run service"
  type        = string
  default     = "simple-billing"
}

variable "frontend_service_name" {
  description = "Name of the frontend Cloud Run service"
  type        = string
  default     = "simple-relay-frontend"
}

variable "firestore_database_name" {
  description = "Firestore database name"
  type        = string
}



variable "api_base_url" {
  description = "API base URL"
  type        = string
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

variable "resend_api_key" {
  description = "Resend API Key for email service"
  type        = string
  sensitive   = true
}

variable "resend_from_email" {
  description = "Resend From Email address"
  type        = string
  default     = "noreply@aifastlane.net"
}

variable "deploy_environment" {
  description = "Environment (production or staging)"
  type        = string
  validation {
    condition     = contains(["production", "staging"], var.deploy_environment)
    error_message = "Environment must be either 'production' or 'staging'."
  }
}

