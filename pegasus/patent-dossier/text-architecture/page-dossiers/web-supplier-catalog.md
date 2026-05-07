# Technical Patent Architecture: Layoutzones

Source Document: page-dossiers/web-supplier-catalog.md
Generated At: 2026-05-07T14:16:57.471Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - subtitle: Catalog Injection, Product Ledger and Promotional Routing
- - product table with image cell, category chip, price, VU, block, MOQ-step, status, truncated SKU, actions
- - Page fetches /v1/supplier/products and /v1/supplier/profile in parallel

## System Architecture
- Implementation Anchor: apps/admin-portal/app/supplier/catalog/page.ts
- **Zoneid:** header
- **Position:** top full-width

## Feature Set
1. Contents
2. Steps
3. Readendpoints
4. Writeendpoints

## Algorithmic and Logical Flow
1. **Flowid:** catalog-bootstrap
2. page spinner loading state
3. error card state
4. ledger empty state with forms still visible
5. ledger with image thumbnails and category chips
6. row-level toggle pending state
7. edit-product modal open
8. edit-product modal with upload preview
9. edit-product modal saving state

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- Endpoint: /v1/catalog/platform-categories
- Endpoint: /v1/supplier/products
- Endpoint: /v1/supplier/products/upload-ticket
- Endpoint: /v1/supplier/products/{sku_id}
- Endpoint: /v1/supplier/profile
- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories
- /v1/supplier/products/{sku_id}
- /v1/supplier/products/upload-ticket
- **Refreshmodel:** initial fetch on mount plus targeted reload after edit and status-toggle actions

## Operational Constraints and State Rules
- page spinner loading state
- error card state
- ledger empty state with forms still visible
- ledger with image thumbnails and category chips
- row-level toggle pending state
- edit-product modal open
- edit-product modal with upload preview
- edit-product modal saving state

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps; Readendpoints; Writeendpoints.
2. Algorithmic sequence includes **Flowid:** catalog-bootstrap | page spinner loading state | error card state.
3. Contract surface is exposed through /v1/catalog/platform-categories, /v1/supplier/products, /v1/supplier/products/upload-ticket, /v1/supplier/products/{sku_id}, /v1/supplier/profile.
4. Integrity constraints include page spinner loading state; error card state; ledger empty state with forms still visible; ledger with image thumbnails and category chips.
