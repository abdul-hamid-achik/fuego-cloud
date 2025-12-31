variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key for server access"
  type        = string
}

variable "environment" {
  description = "Environment name (production, staging)"
  type        = string
  default     = "production"
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "nbg1"

  validation {
    condition     = contains(["nbg1", "fsn1", "hel1", "ash", "hil"], var.location)
    error_message = "Location must be one of: nbg1, fsn1, hel1, ash, hil."
  }
}

variable "server_type" {
  description = "Server type for K3s nodes"
  type        = string
  default     = "cx22"

  validation {
    condition     = contains(["cx22", "cx32", "cx42", "cx52", "cpx21", "cpx31", "cpx41", "cpx51"], var.server_type)
    error_message = "Server type must be a valid Hetzner Cloud server type."
  }
}

variable "node_count" {
  description = "Number of K3s nodes"
  type        = number
  default     = 3

  validation {
    condition     = var.node_count >= 1 && var.node_count <= 10
    error_message = "Node count must be between 1 and 10."
  }
}
