terraform {
  required_version = ">= 1.0"
  
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Pub/Sub Topic for work units
resource "google_pubsub_topic" "work_queue" {
  name = var.topic_name

  message_retention_duration = "86400s" # 24 hours

  labels = {
    environment = var.environment
    service     = "job-driver"
  }
}

# Pub/Sub Subscription for executors
resource "google_pubsub_subscription" "work_queue_sub" {
  name  = "${var.topic_name}-sub"
  topic = google_pubsub_topic.work_queue.name

  # Acknowledgement deadline: 10 minutes for measure execution
  ack_deadline_seconds = 600

  # Retain unacknowledged messages for 7 days
  message_retention_duration = "604800s"

  # Exponential backoff for retries
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  # Dead letter policy after 5 failed attempts
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.work_queue_dlq.id
    max_delivery_attempts = 5
  }

  labels = {
    environment = var.environment
    service     = "job-driver"
  }
}

# Dead Letter Queue for failed work units
resource "google_pubsub_topic" "work_queue_dlq" {
  name = "${var.topic_name}-dlq"

  labels = {
    environment = var.environment
    service     = "job-driver"
  }
}

# Cloud Run Service
resource "google_cloud_run_service" "job_driver" {
  name     = "job-driver"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image

        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }

        env {
          name  = "DISTRIBUTOR"
          value = "pubsub"
        }

        env {
          name  = "PUBSUB_PROJECT_ID"
          value = var.project_id
        }

        env {
          name  = "PUBSUB_TOPIC_ID"
          value = google_pubsub_topic.work_queue.name
        }

        env {
          name  = "BATCH_SIZE"
          value = var.batch_size
        }

        env {
          name  = "COMPLETION_TTL"
          value = var.completion_ttl
        }

        env {
          name  = "AUTO_START"
          value = var.auto_start
        }

        ports {
          container_port = 8080
        }
      }

      # Allow up to 1 hour for job completion
      timeout_seconds = 3600

      service_account_name = google_service_account.driver.email
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = var.max_instances
        "run.googleapis.com/cpu-throttling" = "false"
      }

      labels = {
        environment = var.environment
        service     = "job-driver"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  autogenerate_revision_name = true
}

# Service Account for Cloud Run
resource "google_service_account" "driver" {
  account_id   = "job-driver"
  display_name = "Job Driver Service Account"
  description  = "Service account for Job Driver to publish to Pub/Sub"
}

# IAM: Pub/Sub Publisher role
resource "google_pubsub_topic_iam_member" "driver_publisher" {
  topic  = google_pubsub_topic.work_queue.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${google_service_account.driver.email}"
}

# IAM: Cloud Storage Object Viewer (for reading patient bundles and measures)
resource "google_project_iam_member" "driver_storage_viewer" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "serviceAccount:${google_service_account.driver.email}"
}

# Allow public invocation (optional, can restrict to specific invokers)
resource "google_cloud_run_service_iam_member" "public_invoker" {
  count = var.allow_public_access ? 1 : 0

  service  = google_cloud_run_service.job_driver.name
  location = google_cloud_run_service.job_driver.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Outputs
output "service_url" {
  description = "URL of the deployed Cloud Run service"
  value       = google_cloud_run_service.job_driver.status[0].url
}

output "topic_name" {
  description = "Name of the Pub/Sub work queue topic"
  value       = google_pubsub_topic.work_queue.name
}

output "subscription_name" {
  description = "Name of the Pub/Sub work queue subscription"
  value       = google_pubsub_subscription.work_queue_sub.name
}

output "service_account_email" {
  description = "Email of the service account used by the driver"
  value       = google_service_account.driver.email
}
