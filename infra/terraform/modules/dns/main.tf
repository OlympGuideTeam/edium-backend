resource "yandex_dns_zone" "this" {
  name             = var.zone_name
  zone             = "${var.domain}."
  public           = true
  private_networks = []
}

resource "yandex_dns_recordset" "records" {
  for_each = var.records

  zone_id = yandex_dns_zone.this.id
  name    = each.key
  type    = each.value.type
  ttl     = each.value.ttl
  data    = each.value.data
}
