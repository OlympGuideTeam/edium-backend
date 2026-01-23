locals {
  topics = {
    "user.created" = {
      partitions = 3
      config = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "1209600000" # 14 дней
      }
    }

    "user.deleted" = {
      partitions = 3
      config = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "1209600000" # 14 дней
      }
    }

    "otp.sent" = {
      partitions = 3
      config = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 дней
      }
    }
  }
}

resource "kafka_topic" "topics" {
  for_each = local.topics

  name               = each.key
  partitions         = each.value.partitions
  replication_factor = 1

  config = each.value.config
}