# Marketing Studio Hooks And Settings

Marketing Studio setup items are optional reusable context for `marketing_studio_video`.

- **Hook** (`--hook_id`) sets the opening angle / ad hook. The hook prompt is prepended to the user prompt; it does not replace `--prompt`.
- **Setting** (`--setting_id`) sets the scene or environment context.
- Supported by `marketing_studio_video` only. Do not pass setup items to `marketing_studio_image`.
- **Mode whitelist.** Setup items are valid only for these `--mode` values: `ugc`, `ugc_how_to`, `ugc_unboxing`, `product_review`, `ugc_virtual_try_on`. For `product_showcase`, `tv_spot`, `wild_card`, `virtual_try_on` — do not pass `--hook_id` / `--setting_id`. See `marketing-modes.md` for the full table.
- **Mutually exclusive with ad references.** If the user is generating from an ad reference video, do **not** also pass `--hook_id` / `--setting_id`. The two paths (reference-driven vs composed-from-blocks) cannot be combined.

## Discover Items

```bash
higgsfield marketing-studio hooks list
higgsfield marketing-studio settings list
```

Use `--json` when the agent needs IDs:

```bash
higgsfield marketing-studio hooks list --json
higgsfield marketing-studio settings list --json
```

Filter large lists with search:

```bash
higgsfield marketing-studio hooks list --search sale --json
higgsfield marketing-studio settings list --search office --json
```

The response shape is:

- `items`: setup items with `id`, `name`, `prompt`, `source`, optional `type`, optional media URLs, and pin/status metadata.
- `cursor`: cursor for the next page.
- `has_more`: whether another page exists.

## Generate With Setup Items

Pass one or both IDs:

```bash
PRODUCT_IDS_JSON=$(mktemp)
printf '["<product_id>"]' > "$PRODUCT_IDS_JSON"

higgsfield generate create marketing_studio_video \
  --prompt "..." \
  --mode ugc \
  --product_ids @"$PRODUCT_IDS_JSON" \
  --hook_id <hook_id> \
  --setting_id <setting_id> \
  --duration 15 \
  --aspect_ratio 9:16 \
  --wait
```

When using `--hook_id`, pass product context whenever possible. Hooks are designed to transition into a product pitch and are weak without `product_ids`.

`--mode` is optional; it defaults to `ugc`. Pass it only when the user wants a specific non-default style.

For UGC modes, `--avatars` is optional if the brief clearly mentions a person; the backend can synthesize a Soul Character. Pass `--avatars` when the user selected a specific presenter.

If the CLI returns `Unknown params: hook_id` or `Unknown params: setting_id`, do not retry with that flag for the selected `job_set_type`; its schema does not support setup items.


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
