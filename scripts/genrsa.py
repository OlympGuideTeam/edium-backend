#!/usr/bin/env python3
"""Генерирует RSA-ключ и записывает JWT_RSA_PRIVATE_KEY в .env."""
import re
import os
import subprocess
import sys

env_path = os.path.join(os.path.dirname(__file__), "..", ".env")
env_path = os.path.normpath(env_path)

result = subprocess.run(
    ["openssl", "genpkey", "-algorithm", "RSA", "-pkeyopt", "rsa_keygen_bits:2048"],
    capture_output=True,
    text=True,
)
if result.returncode != 0:
    print("ошибка генерации ключа:", result.stderr, file=sys.stderr)
    sys.exit(1)

key_line = "JWT_RSA_PRIVATE_KEY=" + result.stdout.strip().replace("\n", "\\n")

if os.path.exists(env_path):
    content = open(env_path).read()
    if "JWT_RSA_PRIVATE_KEY=" in content:
        content = re.sub(r"^JWT_RSA_PRIVATE_KEY=.*$", lambda _: key_line, content, flags=re.MULTILINE)
    else:
        content += key_line + "\n"
else:
    content = key_line + "\n"

open(env_path, "w").write(content)
print("RSA ключ записан в .env")
