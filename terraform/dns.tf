# DNS Zone for aifastlane.net
resource "google_dns_managed_zone" "aifastlane_zone" {
  name     = "aifastlane-net-${var.deploy_environment}"
  dns_name = var.deploy_environment == "production" ? "aifastlane.net." : "${var.deploy_environment}.aifastlane.net."

  description = "DNS zone for aifastlane.net domain - ${var.deploy_environment}"
}

# Data source for production zone (needed for staging NS delegation)
data "google_dns_managed_zone" "production_zone" {
  count = var.deploy_environment == "staging" ? 1 : 0
  name  = "aifastlane-net-production"
}

# NS delegation record for staging subdomain in production zone
resource "google_dns_record_set" "staging_ns_delegation" {
  count = var.deploy_environment == "staging" ? 1 : 0
  
  name = "staging.aifastlane.net."
  type = "NS"
  ttl  = 300

  managed_zone = data.google_dns_managed_zone.production_zone[0].name
  rrdatas      = google_dns_managed_zone.aifastlane_zone.name_servers
}

# A records pointing to Cloud Run
resource "google_dns_record_set" "aifastlane_a" {
  name = var.deploy_environment == "production" ? "aifastlane.net." : "${var.deploy_environment}.aifastlane.net."
  type = "A"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = [
    "216.239.32.21",
    "216.239.34.21", 
    "216.239.36.21",
    "216.239.38.21"
  ]
}

# AAAA records for IPv6 support
resource "google_dns_record_set" "aifastlane_aaaa" {
  name = var.deploy_environment == "production" ? "aifastlane.net." : "${var.deploy_environment}.aifastlane.net."
  type = "AAAA"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = [
    "2001:4860:4802:32::15",
    "2001:4860:4802:34::15",
    "2001:4860:4802:36::15", 
    "2001:4860:4802:38::15"
  ]
}

# DMARC record for email security
resource "google_dns_record_set" "dmarc" {
  name = var.deploy_environment == "production" ? "_dmarc.aifastlane.net." : "_dmarc.${var.deploy_environment}.aifastlane.net."
  type = "TXT"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = ["\"v=DMARC1; p=none;\""]
}

# Resend domain key for email authentication
resource "google_dns_record_set" "resend_domainkey" {
  name = var.deploy_environment == "production" ? "resend._domainkey.aifastlane.net." : "resend._domainkey.${var.deploy_environment}.aifastlane.net."
  type = "TXT"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = ["\"p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHRm0xIYZUk0jYPgpT126LFrktukRI2apEiWYF7NI3jnalYOkd4WxwFAIaf9kQayVlOsfZmZNFjYuRo61fMSWNJEDe3IqjeZs7S3CK1Fdu76BpTr+XyHHJhetbLUKngZeUP5qWPuodC4TCBeNncJApB+lixekoCE6bxqgldwWWpwIDAQAB\""]
}

# MX record for send subdomain
resource "google_dns_record_set" "send_mx" {
  name = var.deploy_environment == "production" ? "send.aifastlane.net." : "send.${var.deploy_environment}.aifastlane.net."
  type = "MX"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = ["10 feedback-smtp.ap-northeast-1.amazonses.com."]
}

# SPF record for send subdomain
resource "google_dns_record_set" "send_txt" {
  name = var.deploy_environment == "production" ? "send.aifastlane.net." : "send.${var.deploy_environment}.aifastlane.net."
  type = "TXT"
  ttl  = 300

  managed_zone = google_dns_managed_zone.aifastlane_zone.name

  rrdatas = ["\"v=spf1 include:amazonses.com ~all\""]
}

# Cloud Run domain mapping
resource "google_cloud_run_domain_mapping" "aifastlane_domain" {
  location = var.region
  name     = var.deploy_environment == "production" ? "aifastlane.net" : "${var.deploy_environment}.aifastlane.net"

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = "${var.frontend_service_name}-${var.deploy_environment}"
  }
}

# Outputs
output "dns_name_servers" {
  description = "Name servers for the DNS zone"
  value       = google_dns_managed_zone.aifastlane_zone.name_servers
}

output "domain_mapping_status" {
  description = "Status of the domain mapping"
  value       = google_cloud_run_domain_mapping.aifastlane_domain.status
}

output "domain_name" {
  description = "The domain name for this environment"
  value       = var.deploy_environment == "production" ? "aifastlane.net" : "${var.deploy_environment}.aifastlane.net"
}