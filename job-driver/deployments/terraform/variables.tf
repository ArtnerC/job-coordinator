variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "topic_name" {
  description = "Name of the Pub/Sub topic for work queue"
  type        = string
  default     = "work-queue"
}

variable "container_image" {
  description = "Container image URL (e.g., gcr.io/project/job-driver:latest)"
  type        = string
}

variable "batch_size" {
  description = "Default batch size for work units (lines per batch)"
  type        = string
  default     = "500"
}

variable "completion_ttl" {
  description = "Time to keep service running after job completion"
  type        = string
  default     = "10m"
}

variable "auto_start" {
  description = "Whether to automatically start job on service startup"
  type        = string
  default     = "true"
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = string
  default     = "100"
}

variable "allow_public_access" {
  description = "Whether to allow public access to the Cloud Run service"
  type        = bool
  default     = false
}
