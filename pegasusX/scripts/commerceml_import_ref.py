#!/usr/bin/env python3
"""Reference CommerceML 2.x → partner upsert payload converter (Phase 2).

Parses a minimal subset of CommerceML XML (goods / offers / rests) and emits
JSON bodies compatible with:
  PUT /partner/v1/catalog/products
  PUT /partner/v1/catalog/prices
  PUT /partner/v1/inventory/stock

This is intentionally best-effort and not a certified 1C connector.
"""
from __future__ import annotations

import argparse
import json
import sys
import xml.etree.ElementTree as ET
from pathlib import Path


def local(tag: str) -> str:
    if "}" in tag:
        return tag.rsplit("}", 1)[-1]
    return tag


def text(el: ET.Element | None) -> str:
    if el is None or el.text is None:
        return ""
    return el.text.strip()


def find_child(parent: ET.Element, name: str) -> ET.Element | None:
    for c in parent:
        if local(c.tag) == name:
            return c
    return None


def parse_products(path: Path) -> list[dict]:
    root = ET.parse(path).getroot()
    items: list[dict] = []
    for el in root.iter():
        tag = local(el.tag).lower()
        if tag not in ("товар", "product", "good"):
            continue
        iid = text(find_child(el, "Ид")) or text(find_child(el, "Id")) or text(find_child(el, "id"))
        name = text(find_child(el, "Наименование")) or text(find_child(el, "Name")) or text(find_child(el, "name"))
        if not iid or not name:
            continue
        barcode = text(find_child(el, "Штрихкод")) or text(find_child(el, "Barcode"))
        items.append(
            {
                "external_id": iid,
                "name": name,
                "barcode": barcode,
                "currency": "UZS",
                "unit": "ea",
            }
        )
    return items


def parse_prices(path: Path) -> list[dict]:
    root = ET.parse(path).getroot()
    items: list[dict] = []
    for el in root.iter():
        tag = local(el.tag).lower()
        if tag not in ("предложение", "offer"):
            continue
        iid = text(find_child(el, "Ид")) or text(find_child(el, "Id"))
        price_el = find_child(el, "Цены") or find_child(el, "Prices")
        price_minor = 0
        if price_el is not None:
            for p in price_el:
                if local(p.tag).lower() in ("цена", "price"):
                    raw = text(find_child(p, "ЦенаЗаЕдиницу")) or text(find_child(p, "Price")) or text(p)
                    try:
                        # CommerceML prices are major units; convert to minor (×100).
                        price_minor = int(round(float(raw.replace(",", ".")) * 100))
                    except ValueError:
                        price_minor = 0
                    break
        if iid:
            items.append({"external_id": iid, "price_minor": price_minor, "currency": "UZS"})
    return items


def parse_rests(path: Path, warehouse_id: str) -> list[dict]:
    root = ET.parse(path).getroot()
    items: list[dict] = []
    for el in root.iter():
        tag = local(el.tag).lower()
        if tag not in ("остаток", "rest", "stock"):
            continue
        iid = text(find_child(el, "Ид")) or text(find_child(el, "Id"))
        qty_raw = text(find_child(el, "Количество")) or text(find_child(el, "Quantity")) or "0"
        try:
            qty = int(round(float(qty_raw.replace(",", "."))))
        except ValueError:
            qty = 0
        if iid:
            items.append(
                {
                    "external_id": iid,
                    "warehouse_id": warehouse_id,
                    "quantity_on_hand": qty,
                }
            )
    return items


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--import", dest="import_path", help="CommerceML import.xml (goods)")
    ap.add_argument("--offers", dest="offers_path", help="CommerceML offers.xml")
    ap.add_argument("--rests", dest="rests_path", help="CommerceML rests.xml")
    ap.add_argument("--warehouse-id", default="wh-1")
    ap.add_argument("--out", required=True, help="Output JSON path")
    args = ap.parse_args()

    out = {"products": {"items": []}, "prices": {"items": []}, "stock": {"items": []}}
    if args.import_path:
        out["products"]["items"] = parse_products(Path(args.import_path))
    if args.offers_path:
        out["prices"]["items"] = parse_prices(Path(args.offers_path))
    if args.rests_path:
        out["stock"]["items"] = parse_rests(Path(args.rests_path), args.warehouse_id)

    Path(args.out).write_text(json.dumps(out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(
        f"wrote {args.out}: products={len(out['products']['items'])} "
        f"prices={len(out['prices']['items'])} stock={len(out['stock']['items'])}",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
