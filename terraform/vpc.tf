# VPC configuration for private IP setup (conditional)
resource "google_compute_network" "private_network" {
  count                   = var.enable_private_ip ? 1 : 0
  name                    = "${var.service_name}-vpc"
  auto_create_subnetworks = false
  
  depends_on = [google_project_service.required_apis]
}

resource "google_compute_subnetwork" "private_subnet" {
  count         = var.enable_private_ip ? 1 : 0
  name          = "${var.service_name}-subnet"
  ip_cidr_range = "10.0.0.0/24"
  region        = var.region
  network       = google_compute_network.private_network[0].id
}

# Private service connection for Cloud SQL
resource "google_compute_global_address" "private_ip_address" {
  count         = var.enable_private_ip ? 1 : 0
  name          = "${var.service_name}-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.private_network[0].id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  count                   = var.enable_private_ip ? 1 : 0
  network                 = google_compute_network.private_network[0].id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address[0].name]
  
  depends_on = [google_project_service.required_apis]
}

# VPC Access Connector for Cloud Run
resource "google_vpc_access_connector" "connector" {
  count         = var.enable_private_ip ? 1 : 0
  name          = "${var.service_name}-connector"
  region        = var.region
  network       = google_compute_network.private_network[0].name
  ip_cidr_range = "10.1.0.0/28"
  
  min_instances = 2
  max_instances = 3
  
  depends_on = [google_project_service.required_apis]
}