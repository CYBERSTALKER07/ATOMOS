# Avatars

## Preset vs Custom

| | Preset | Custom |
|---|---|---|
| Source | Curated by Higgsfield | User-uploaded |
| Cost | None for selection | Cost of upload |
| Diversity | Limited but professional | Unlimited |
| Use when | Generic ad, fast turnaround | Brand-specific face, founder, employee |

## Listing presets

```bash
higgsfield marketing-studio avatars list
higgsfield marketing-studio avatars list --json | jq '.[] | select(.gender=="female")'
```

Filter by `name`, `gender`, etc. on the JSON output.

## Creating a custom avatar

```bash
ID=$(higgsfield upload create founder.png)
URL=$(higgsfield upload create founder.png --json | jq -r .url)   # if you need cloudfront URL
higgsfield marketing-studio avatars create --name "Founder" --image $ID --image-url $URL
```

`--image-url` is the cloudfront URL from the upload. Required by the API.

## Passing to video

```bash
AVATARS_JSON=$(mktemp)
printf '[{"id":"<avatar_id>","type":"preset"}]' > "$AVATARS_JSON"

higgsfield generate create marketing_studio_video \
  --avatars @"$AVATARS_JSON" \
  ... \
  --wait
```

`type` is `preset` for curated, `custom` for user-created.
`--avatars` expects a JSON array, so pass it via `@/path/to/file.json`.

For UGC modes, an avatar is optional if the brief clearly mentions a person and no specific presenter was requested; the backend can synthesize a Soul Character automatically.


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
