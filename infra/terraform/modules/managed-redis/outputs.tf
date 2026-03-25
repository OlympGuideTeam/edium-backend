output "cluster_id" {
  description = "ID кластера Redis"
  value       = yandex_mdb_redis_cluster.this.id
}

output "cluster_hosts" {
  description = "FQDN хостов кластера"
  value       = [for h in yandex_mdb_redis_cluster.this.host : h.fqdn]
}

output "cluster_host" {
  description = "FQDN первого хоста (для подключения)"
  value       = yandex_mdb_redis_cluster.this.host[0].fqdn
}
