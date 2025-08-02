# Cloud Armor Security Policy
resource "google_compute_security_policy" "policy" {
  name    = "${var.policy_name}-${var.environment}"
  project = var.project_id

  # Default rule - allow all traffic
  rule {
    action   = "allow"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "Default rule, higher priority overrides it"
  }

  # General rate limiting rule
  rule {
    action   = "rate_based_ban"
    priority = "1000"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      rate_limit_threshold {
        count        = 20
        interval_sec = 60
      }
      conform_action   = "allow"
      exceed_action    = "deny(429)"
      enforce_on_key   = "IP"
      ban_duration_sec = 60
    }
    description = "Rate limiting: 20 requests per minute per IP, 1 minute ban"
  }

  # API burst limiting for API endpoints
  rule {
    action   = "rate_based_ban"
    priority = "1100"
    match {
      expr {
        expression = "request.path.matches('/api/.*')"
      }
    }
    rate_limit_options {
      rate_limit_threshold {
        count        = 10
        interval_sec = 30
      }
      conform_action   = "allow"
      exceed_action    = "deny(429)"
      enforce_on_key   = "IP"
      ban_duration_sec = 60
    }
    description = "API burst limiting: 10 requests per 30 seconds per IP, 1 minute ban"
  }

  # Login endpoint rate limiting
  rule {
    action   = "rate_based_ban"
    priority = "1200"
    match {
      expr {
        expression = "request.path.matches('/api/v1/auth/.*')"
      }
    }
    rate_limit_options {
      rate_limit_threshold {
        count        = 5
        interval_sec = 60
      }
      conform_action   = "allow"
      exceed_action    = "deny(429)"
      enforce_on_key   = "IP"
      ban_duration_sec = 60
    }
    description = "Login endpoint limiting: 5 attempts per minute per IP, 1 minute ban"
  }

  # Block known malicious IPs (only if IPs are provided)
  dynamic "rule" {
    for_each = length(var.blocked_ips) > 0 ? [1] : []
    content {
      action   = "deny(403)"
      priority = "1300"
      match {
        versioned_expr = "SRC_IPS_V1"
        config {
          src_ip_ranges = var.blocked_ips
        }
      }
      description = "Block known malicious IP addresses"
    }
  }

  # Block SQL injection attempts
  rule {
    action   = "deny(403)"
    priority = "1400"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-stable')"
      }
    }
    description = "Block SQL injection attempts"
  }

  # Block XSS attempts
  rule {
    action   = "deny(403)"
    priority = "1500"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('xss-stable')"
      }
    }
    description = "Block XSS attempts"
  }

  # Block LFI/RFI attempts
  rule {
    action   = "deny(403)"
    priority = "1600"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('lfi-stable')"
      }
    }
    description = "Block Local File Inclusion (LFI) attempts"
  }

  # Block RCE attempts
  rule {
    action   = "deny(403)"
    priority = "1700"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('rce-stable')"
      }
    }
    description = "Block Remote Code Execution (RCE) attempts"
  }

  # Block method abuse
  rule {
    action   = "deny(403)"
    priority = "1800"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('methodenforcement-stable')"
      }
    }
    description = "Block method abuse"
  }

  # Block scanner detection
  rule {
    action   = "deny(403)"
    priority = "1900"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('scannerdetection-stable')"
      }
    }
    description = "Block scanner detection"
  }

  # Block protocol attacks
  rule {
    action   = "deny(403)"
    priority = "2000"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('protocolattack-stable')"
      }
    }
    description = "Block protocol attacks"
  }

  # Block PHP injection attempts
  rule {
    action   = "deny(403)"
    priority = "2100"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('php-stable')"
      }
    }
    description = "Block PHP injection attempts"
  }

  # Custom rules from variables
  dynamic "rule" {
    for_each = var.custom_rules
    content {
      action   = rule.value.action
      priority = rule.value.priority
      match {
        dynamic "expr" {
          for_each = rule.value.match_expression != null ? [rule.value.match_expression] : []
          content {
            expression = expr.value
          }
        }
        dynamic "expr" {
          for_each = rule.value.match_versioned_expr != null ? [rule.value.match_versioned_expr] : []
          content {
            expression = expr.value
          }
        }
      }
      description = rule.value.description
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Attach Cloud Armor policy to backend service
resource "google_compute_backend_service" "backend_with_armor" {
  name        = "${var.backend_service_name}-with-armor-${var.environment}"
  project     = var.project_id
  protocol    = "HTTP"
  port_name   = "http"
  timeout_sec = 30

  backend {
    group = var.backend_service_group
  }

  # Health checks removed - not supported with Serverless NEGs for Cloud Run

  security_policy = google_compute_security_policy.policy.name

  log_config {
    enable = true
  }

  lifecycle {
    prevent_destroy = true
  }
} 