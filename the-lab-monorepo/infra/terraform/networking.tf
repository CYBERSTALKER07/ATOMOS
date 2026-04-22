# ──────────────────────────────────────────────────────────────────────────────
# networking.tf — Global networking for the V.O.I.D. platform.
#
# Resources:
#   1. Global External Application Load Balancer (anycast IP)
#   2. Cloud Armor security policy — DDoS + OWASP CRS WAF + per-IP rate limit
#   3. Cloud CDN — cache catalog/product read endpoints at edge
#   4. Private Service Connect endpoints — Spanner + Redis never leave VPC
#   5. Cloud NAT — egress for GKE nodes (no public node IPs needed)
# ──────────────────────────────────────────────────────────────────────────────

# ── 1. Static global anycast IP ───────────────────────────────────────────

resource "google_compute_global_address" "void_frontend" {
  name         = "void-frontend-ip"
  project      = var.project_id
  address_type = "EXTERNAL"
  ip_version   = "IPV4"
  description  = "Anycast IP for the V.O.I.D. Global Load Balancer."
}

# ── 2. Cloud Armor security policy ────────────────────────────────────────

resource "google_compute_security_policy" "void_armor" {
  name        = "void-armor"
  project     = var.project_id
  description = "Cloud Armor WAF + DDoS + rate limiting for V.O.I.D. backend."

  # ── Rule 1000: throttle individual IPs to 1000 req/min ──────────────────
  rule {
    action   = "throttle"
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      rate_limit_threshold {
        count        = 1000
        interval_sec = 60
      }
      enforce_on_key = "IP"
    }
    description = "Per-IP rate limit: 1000 req/min"
  }

  # ── Rule 2000: OWASP CRS 3.3 — SQLi ───────────────────────────────────
  rule {
    action   = "deny(403)"
    priority = 2000
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable')"
      }
    }
    description = "Block SQL injection (OWASP CRS 3.3)"
  }

  # ── Rule 2001: OWASP CRS 3.3 — XSS ────────────────────────────────────
  rule {
    action   = "deny(403)"
    priority = 2001
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('xss-v33-stable')"
      }
    }
    description = "Block XSS (OWASP CRS 3.3)"
  }

  # ── Rule 2002: OWASP CRS — Remote File Inclusion ───────────────────────
  rule {
    action   = "deny(403)"
    priority = 2002
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('rfi-canary')"
      }
    }
    description = "Block RFI (OWASP CRS)"
  }

  # ── Rule 3000: Allow internal health-check probers ─────────────────────
  rule {
    action   = "allow"
    priority = 3000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        # GFE health-check probe ranges per https://cloud.google.com/load-balancing/docs/health-check-concepts
        src_ip_ranges = ["35.191.0.0/16", "130.211.0.0/22"]
      }
    }
    description = "Allow GFE health-check prober IPs"
  }

  # ── Rule 9999: default allow ───────────────────────────────────────────
  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default allow — upper-tier rules handle rejection"
  }
}

# ── 3. GKE NEG (Network Endpoint Group) — backend service ────────────────
# The NEG is created automatically by the GKE Ingress controller when the
# backend Service is annotated with cloud.google.com/neg: '{"ingress":true}'.
# We reference it here by naming convention.

resource "google_compute_backend_service" "void_backend" {
  name                  = "void-backend"
  project               = var.project_id
  protocol              = "HTTP2"
  port_name             = "http"
  timeout_sec           = 30
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.void_armor.id

  # CDN is enabled only for safe read endpoints; POST/PUT/DELETE are
  # bypassed by the CDN by default (non-GET/HEAD are always cache-miss).
  enable_cdn = true
  cdn_policy {
    cache_mode                   = "CACHE_ALL_STATIC"
    default_ttl                  = 300  # 5 min for catalog/product reads
    max_ttl                      = 3600 # 1 hour ceiling
    client_ttl                   = 300
    serve_while_stale            = 120
    negative_caching             = true
    signed_url_cache_max_age_sec = 0
    cache_key_policy {
      include_host         = true
      include_protocol     = true
      include_query_string = true
    }
  }

  # Health check for the GKE pods (matches /healthz route)
  health_checks = [google_compute_health_check.void_http.id]

  # ── Session affinity — Maglev ring-hash keyed on supplier ID ─────────────
  # When a supplier has an active WebSocket connection or warm in-process
  # L1 cache, subsequent requests from the same supplier should land on the
  # same backend pod.  GCP implements this with a consistent hash over the
  # X-Supplier-Id header using a Maglev-style lookup table at the LB layer:
  # - Zero coordination between pods — each LB replica independently computes
  #   the same slot-to-pod assignment.
  # - When a pod is added or removed, only ~1/N suppliers are re-routed
  #   (vs all of them with naive least-connection).
  # - WebSocket upgrades inherit the same affinity, so reconnects land on the
  #   same pod and the hub subscription is already warm.
  #
  # Falls back to round-robin automatically when X-Supplier-Id is absent
  # (e.g. public catalogue reads, unauthenticated health checks).
  session_affinity  = "HEADER_FIELD"
  locality_lb_policy = "RING_HASH"

  consistent_hash {
    http_header_name  = "X-Supplier-Id"
    minimum_ring_size = 1024
  }

  log_config {
    enable      = true
    sample_rate = 1.0
  }
}

resource "google_compute_health_check" "void_http" {
  name    = "void-http-hc"
  project = var.project_id

  http_health_check {
    request_path = "/healthz"
    port         = 8080
  }

  check_interval_sec  = 10
  timeout_sec         = 5
  healthy_threshold   = 1
  unhealthy_threshold = 3
}

# ── 4. URL map + forwarding rules (HTTPS redirect included) ───────────────

resource "google_compute_url_map" "void_http_redirect" {
  name    = "void-http-redirect"
  project = var.project_id

  default_url_redirect {
    https_redirect         = true
    strip_query            = false
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
  }
}

resource "google_compute_target_http_proxy" "void_redirect" {
  name    = "void-http-redirect-proxy"
  project = var.project_id
  url_map = google_compute_url_map.void_http_redirect.id
}

resource "google_compute_global_forwarding_rule" "void_http" {
  name                  = "void-http-fwd"
  project               = var.project_id
  target                = google_compute_target_http_proxy.void_redirect.id
  ip_address            = google_compute_global_address.void_frontend.address
  port_range            = "80"
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

resource "google_compute_url_map" "void_https" {
  name    = "void-https"
  project = var.project_id

  default_service = google_compute_backend_service.void_backend.id

  # Cache static catalog reads at the edge (CDN bypass for everything else)
  host_rule {
    hosts        = [var.backend_hostname]
    path_matcher = "api"
  }

  path_matcher {
    name            = "api"
    default_service = google_compute_backend_service.void_backend.id

    # These prefixes are cacheable GET endpoints
    path_rule {
      paths   = ["/v1/catalog/*", "/v1/products/*", "/v1/categories/*"]
      service = google_compute_backend_service.void_backend.id
    }
  }
}

resource "google_compute_managed_ssl_certificate" "void_cert" {
  name    = "void-managed-cert"
  project = var.project_id

  managed {
    domains = [var.backend_hostname]
  }
}

resource "google_compute_target_https_proxy" "void_https" {
  name             = "void-https-proxy"
  project          = var.project_id
  url_map          = google_compute_url_map.void_https.id
  ssl_certificates = [google_compute_managed_ssl_certificate.void_cert.id]
}

resource "google_compute_global_forwarding_rule" "void_https" {
  name                  = "void-https-fwd"
  project               = var.project_id
  target                = google_compute_target_https_proxy.void_https.id
  ip_address            = google_compute_global_address.void_frontend.address
  port_range            = "443"
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

# ── 5. Private Service Connect — Spanner ──────────────────────────────────
# Routes Spanner API calls from GKE pods through VPC-internal PSC endpoints
# so no Spanner traffic traverses the public internet.

resource "google_compute_global_address" "spanner_psc" {
  name          = "void-spanner-psc"
  project       = var.project_id
  purpose       = "PRIVATE_SERVICE_CONNECT"
  network       = var.vpc_network
  address_type  = "INTERNAL"
  address       = "10.100.0.2"
  prefix_length = 32
}

resource "google_compute_global_forwarding_rule" "spanner_psc" {
  name                  = "void-spanner-psc-fwd"
  project               = var.project_id
  target                = "all-apis"
  ip_address            = google_compute_global_address.spanner_psc.id
  network               = var.vpc_network
  load_balancing_scheme = ""
}

# ── 6. Cloud NAT — controlled egress for GKE nodes ────────────────────────

resource "google_compute_router" "void_nat_router" {
  name    = "void-nat-router"
  project = var.project_id
  region  = var.region
  network = var.vpc_network
}

resource "google_compute_router_nat" "void_nat" {
  name                               = "void-nat"
  project                            = var.project_id
  region                             = var.region
  router                             = google_compute_router.void_nat_router.name
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# ── Variables used by networking.tf ───────────────────────────────────────

variable "vpc_network" {
  type        = string
  description = "Self-link or name of the VPC network."
  default     = "default"
}
