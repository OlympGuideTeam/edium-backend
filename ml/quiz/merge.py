#!/usr/bin/env python3
"""Склеивает несколько JSONL в один файл (по порядку аргументов, без дедупликации)."""
import argparse
import shutil


def main():
    p = argparse.ArgumentParser(description="Объединить dataset-1.jsonl dataset-2.jsonl ... → один файл")
    p.add_argument("inputs", nargs="+", help="Входные .jsonl")
    p.add_argument("-o", "--output", required=True, help="Выходной файл")
    args = p.parse_args()

    with open(args.output, "wb") as out:
        for path in args.inputs:
            with open(path, "rb") as f:
                shutil.copyfileobj(f, out)
    print(f"Записано: {args.output} ({len(args.inputs)} файлов)")


if __name__ == "__main__":
    main()
