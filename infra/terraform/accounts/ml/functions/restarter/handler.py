"""
Cloud Function: авто-рестарт прерываемой GPU VM Sphinx.

Запускается по таймеру каждые 5 минут. Если VM в статусе STOPPED или
CRASHED — вызывает Start. Остальные статусы (RUNNING, STARTING и т.д.)
игнорируются.

Переменные окружения:
    VM_ID — ID инстанса GPU VM (задаётся через Terraform)
"""

import os

import yandexcloud
from yandex.cloud.compute.v1 import instance_service_pb2 as is_pb2
from yandex.cloud.compute.v1 import instance_service_pb2_grpc as is_grpc

# Статусы из compute/v1/instance.proto
STATUS_STOPPED = 4
STATUS_CRASHED = 9


def handler(event, context):
    vm_id = os.environ["VM_ID"]

    sdk = yandexcloud.SDK()
    stub = sdk.client(is_grpc.InstanceServiceStub)

    instance = stub.Get(is_pb2.GetInstanceRequest(instance_id=vm_id))
    status = instance.status

    if status in (STATUS_STOPPED, STATUS_CRASHED):
        stub.Start(is_pb2.StartInstanceRequest(instance_id=vm_id))
        print(f"VM {vm_id}: статус {status} → запущена")
        return {"statusCode": 200, "body": "started"}

    print(f"VM {vm_id}: статус {status}, действие не требуется")
    return {"statusCode": 200, "body": "ok"}
