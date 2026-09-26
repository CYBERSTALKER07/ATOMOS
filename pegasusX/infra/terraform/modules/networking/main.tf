/**
 * Module: Networking
 * Core VPC, Subnets, Cloud Router, Cloud NAT (Deterministic Egress), PSA Peering, and Firewalls
 */

# 1. Custom VPC Network
resource "google_compute_network" "vpc" {
  name                    = var.network_name
  auto_create_subnetworks = false
  routing_mode            = "GLOBAL"
  project                 = var.project_id
  description             = "PegasusX Enterprise isolated VPC network"
}

# 2. Regional Subnetwork with Secondary IP Ranges for GKE Pods and Services
resource "google_compute_subnetwork" "subnet" {
  name                     = var.subnet_name
  ip_cidr_range            = var.primary_cidr_block
  region                   = var.region
  network                  = google_compute_network.vpc.id
  project                  = var.project_id
  private_ip_google_access = true
  description              = "Primary subnet for GKE nodes, VMs, and private workloads"

  secondary_ip_range {
    range_name    = var.pod_ip_range_name
    ip_cidr_range = var.pod_cidr_block
  }

  secondary_ip_range {
    range_name    = var.service_ip_range_name
    ip_cidr_range = var.service_cidr_block
  }

  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling        = 0.5
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

# 3. Static External IP Addresses for Deterministic Cloud NAT Egress (Soliq OFD & Banking Allowlist)
resource "google_compute_address" "nat_ips" {
  count        = 2
  name         = "${var.network_name}-nat-egress-ip-${count.index + 1}"
  region       = var.region
  project      = var.project_id
  address_type = "EXTERNAL"
  network_tier = "PREMIUM"
  description  = "Dedicated static outbound IP address ${count.index + 1} for Soliq OFD and banking whitelist"
}

# 4. Cloud Router for NAT Gateway
resource "google_compute_router" "router" {
  name        = "${var.network_name}-router"
  region      = var.region
  network     = google_compute_network.vpc.id
  project     = var.project_id
  description = "Regional Cloud Router for PegasusX Cloud NAT"

  bgp {
    asn = 64514
  }
}

# 5. Cloud NAT Gateway for Deterministic Egress
resource "google_compute_router_nat" "nat_gateway" {
  name                               = "${var.network_name}-nat-gw"
  router                             = google_compute_router.router.name
  region                             = var.region
  project                            = var.project_id
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = google_compute_address.nat_ips[*].self_link
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  min_ports_per_vm                    = 64
  max_ports_per_vm                    = 2048
  enable_endpoint_independent_mapping = true

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# 6. Private Service Access (PSA) Global Internal IP Allocation for Cloud Memorystore Redis & Managed Services
resource "google_compute_global_address" "psa_range" {
  name          = "${var.network_name}-psa-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = var.psa_prefix_length
  network       = google_compute_network.vpc.id
  project       = var.project_id
  description   = "Reserved internal IP block for Private Service Access (Redis / Service Networking)"
}

# 7. Service Networking Peering Connection
resource "google_service_networking_connection" "psa_peering" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.psa_range.name]
  deletion_policy         = "ABANDON"
}

# 8. Firewall Rules

# 8.1 Internal VPC & Secondary Pod/Service Communication
resource "google_compute_firewall" "allow_internal" {
  name        = "${var.network_name}-allow-internal"
  network     = google_compute_network.vpc.name
  project     = var.project_id
  description = "Allow all internal TCP/UDP/ICMP traffic between nodes, pods, and services"
  priority    = 1000
  direction   = "INGRESS"

  source_ranges = [
    var.primary_cidr_block,
    var.pod_cidr_block,
    var.service_cidr_block
  ]

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }
}

# 8.2 Allow Google Cloud Load Balancer & Health Checks
resource "google_compute_firewall" "allow_health_checks" {
  name        = "${var.network_name}-allow-health-checks"
  network     = google_compute_network.vpc.name
  project     = var.project_id
  description = "Allow Google Cloud health checks (GCLB / Ingress)"
  priority    = 1000
  direction   = "INGRESS"

  source_ranges = [
    "35.191.0.0/16",
    "130.211.0.0/22"
  ]

  allow {
    protocol = "tcp"
    ports    = ["80", "443", "8080", "8081", "8082", "50055"]
  }
}

# 8.3 Allow GKE Control Plane (Master) to Node Kubelet & Webhook Admission Controllers
resource "google_compute_firewall" "allow_gke_master" {
  name        = "${var.network_name}-allow-gke-master"
  network     = google_compute_network.vpc.name
  project     = var.project_id
  description = "Allow GKE control plane to access worker nodes and webhook controllers"
  priority    = 1000
  direction   = "INGRESS"

  source_ranges = [
    var.gke_master_cidr_block
  ]

  allow {
    protocol = "tcp"
    ports    = ["443", "10250", "8080", "8081", "8082", "8443", "9443"]
  }
}

# 8.4 Default Low-Priority Deny-All Ingress for Regulatory Compliance
resource "google_compute_firewall" "deny_all_ingress" {
  name        = "${var.network_name}-deny-all-ingress"
  network     = google_compute_network.vpc.name
  project     = var.project_id
  description = "Audit compliance rule: default deny all unmatched inbound traffic"
  priority    = 65534
  direction   = "INGRESS"

  source_ranges = ["0.0.0.0/0"]

  deny {
    protocol = "all"
  }
}
