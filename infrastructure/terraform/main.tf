terraform {
  required_version = ">= 1.0"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.45"
    }
  }

  # Using local state for now
  # backend "s3" {
  #   bucket                      = "nexo-cloud-terraform-state"
  #   key                         = "infrastructure/terraform.tfstate"
  #   region                      = "auto"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   skip_requesting_account_id  = true
  #   skip_s3_checksum            = true
  #   endpoints = {
  #     s3 = "https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com"
  #   }
  # }
}

provider "hcloud" {
  token = var.hcloud_token
}

resource "hcloud_ssh_key" "nexo" {
  name       = "nexo-${var.environment}-key"
  public_key = var.ssh_public_key
}

resource "hcloud_network" "nexo" {
  name     = "nexo-${var.environment}-network"
  ip_range = "10.0.0.0/16"
}

resource "hcloud_network_subnet" "nexo" {
  network_id   = hcloud_network.nexo.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = "10.0.1.0/24"
}

resource "hcloud_firewall" "nexo" {
  name = "nexo-${var.environment}-firewall"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "6443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "10250"
    source_ips = ["10.0.0.0/16"]
  }

  rule {
    direction  = "in"
    protocol   = "udp"
    port       = "8472"
    source_ips = ["10.0.0.0/16"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "2379-2380"
    source_ips = ["10.0.0.0/16"]
  }

  # NodePort range for Traefik (allow from load balancer private network)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "30080"
    source_ips = ["10.0.0.0/16", "0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "30443"
    source_ips = ["10.0.0.0/16", "0.0.0.0/0", "::/0"]
  }
}

resource "hcloud_placement_group" "nexo" {
  name = "nexo-${var.environment}-placement"
  type = "spread"
}

resource "hcloud_server" "nodes" {
  count              = var.node_count
  name               = "nexo-${var.environment}-node-${count.index + 1}"
  server_type        = var.server_type
  image              = "ubuntu-24.04"
  location           = var.location
  ssh_keys           = [hcloud_ssh_key.nexo.id]
  placement_group_id = hcloud_placement_group.nexo.id
  firewall_ids       = [hcloud_firewall.nexo.id]

  labels = {
    environment = var.environment
    role        = count.index == 0 ? "k3s-master" : "k3s-worker"
  }

  user_data = <<-EOF
    #!/bin/bash
    set -euo pipefail

    apt-get update && apt-get upgrade -y
    apt-get install -y curl wget apt-transport-https ca-certificates gnupg lsb-release

    swapoff -a
    sed -i '/swap/d' /etc/fstab

    cat <<EOL | tee /etc/modules-load.d/k8s.conf
    overlay
    br_netfilter
    EOL

    modprobe overlay
    modprobe br_netfilter

    cat <<EOL | tee /etc/sysctl.d/k8s.conf
    net.bridge.bridge-nf-call-iptables  = 1
    net.bridge.bridge-nf-call-ip6tables = 1
    net.ipv4.ip_forward                 = 1
    EOL

    sysctl --system

    touch /tmp/node-ready
  EOF
}

resource "hcloud_server_network" "nodes" {
  count     = var.node_count
  server_id = hcloud_server.nodes[count.index].id
  network_id = hcloud_network.nexo.id
  ip        = "10.0.1.${10 + count.index}"

  depends_on = [hcloud_network_subnet.nexo]
}

resource "hcloud_load_balancer" "nexo" {
  name               = "nexo-${var.environment}-lb"
  load_balancer_type = "lb11"
  location           = var.location

  labels = {
    environment = var.environment
    role        = "ingress"
  }
}

resource "hcloud_load_balancer_network" "nexo" {
  load_balancer_id = hcloud_load_balancer.nexo.id
  network_id       = hcloud_network.nexo.id
  ip               = "10.0.1.100"

  depends_on = [hcloud_network_subnet.nexo]
}

resource "hcloud_load_balancer_target" "nodes" {
  count            = var.node_count
  type             = "server"
  load_balancer_id = hcloud_load_balancer.nexo.id
  server_id        = hcloud_server.nodes[count.index].id
  use_private_ip   = true

  depends_on = [hcloud_load_balancer_network.nexo, hcloud_server_network.nodes]
}

resource "hcloud_load_balancer_service" "http" {
  load_balancer_id = hcloud_load_balancer.nexo.id
  protocol         = "tcp"
  listen_port      = 80
  destination_port = 30080

  health_check {
    protocol = "tcp"
    port     = 30080
    interval = 10
    timeout  = 5
    retries  = 3
  }
}

resource "hcloud_load_balancer_service" "https" {
  load_balancer_id = hcloud_load_balancer.nexo.id
  protocol         = "tcp"
  listen_port      = 443
  destination_port = 30443

  health_check {
    protocol = "tcp"
    port     = 30443
    interval = 10
    timeout  = 5
    retries  = 3
  }
}
