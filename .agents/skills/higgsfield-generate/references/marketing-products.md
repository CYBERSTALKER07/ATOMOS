# Products

Two ways to register a product: URL fetch (auto-imports title, description, images) or manual (provide your own).

## URL fetch (default)

```bash
ID=$(higgsfield marketing-studio products fetch --url https://shop.example.com/sneakers --wait --json | jq -r .id)
```

`--wait` polls until `status` is `completed` or `failed`. Default timeout 90s.

If `failed`, check `fail_reason` — usually invalid URL or scrape blocked.

App Store URLs auto-route to `webproducts` (different endpoint). Use:

```bash
higgsfield marketing-studio webproducts fetch --url https://apps.apple.com/... --wait
```

## Manual

When the user has product photos and details:

```bash
A=$(higgsfield upload create shoe1.png)
B=$(higgsfield upload create shoe2.png)
higgsfield marketing-studio products create \
  --title "AeroRun Pro" \
  --description "Lightweight running shoe" \
  --image $A --image $B
```

Returns the product entity directly (no polling needed).

## Manual webproduct

For App Store / web pages without URL fetch:

```bash
higgsfield marketing-studio webproducts create \
  --url "https://example.com" \
  --title "MyApp" \
  --subtitle "Productivity for teams" \
  --description "..." \
  --favicon-url "https://example.com/favicon.png" \
  --desktop "https://cdn/screenshot1.png" \
  --mobile "https://cdn/mobile-screenshot.png"
```

## Listing

```bash
higgsfield marketing-studio products list
higgsfield marketing-studio products list --json
higgsfield marketing-studio webproducts list
```


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
