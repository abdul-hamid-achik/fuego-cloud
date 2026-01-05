output "network_id" {
  description = "ID of the private network"
  value       = hcloud_network.nexo.id
}

output "load_balancer_ip" {
  description = "Public IP of the load balancer"
  value       = hcloud_load_balancer.nexo.ipv4
}

output "master_node_ip" {
  description = "Public IP of the master node"
  value       = hcloud_server.nodes[0].ipv4_address
}

output "node_ips" {
  description = "Public IPs of all nodes"
  value       = hcloud_server.nodes[*].ipv4_address
}

output "node_private_ips" {
  description = "Private IPs of all nodes"
  value       = hcloud_server_network.nodes[*].ip
}

output "ansible_inventory" {
  description = "Generated Ansible inventory"
  value = templatefile("${path.module}/templates/inventory.tftpl", {
    master_ip       = hcloud_server.nodes[0].ipv4_address
    master_private  = hcloud_server_network.nodes[0].ip
    worker_ips      = slice(hcloud_server.nodes[*].ipv4_address, 1, length(hcloud_server.nodes))
    worker_privates = slice(hcloud_server_network.nodes[*].ip, 1, length(hcloud_server_network.nodes))
  })
  sensitive = false
}
