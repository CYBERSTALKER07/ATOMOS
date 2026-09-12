# Soul Troubleshooting

## `Minimum Basic plan required`

Soul training needs a paid plan. Tell the user to upgrade.

## `Training failed`

Common causes:

- Too few photos (<5) or too uniform.
- Heavy occlusion (sunglasses, hats).
- Group photos confusing identity.
- Upload type mismatch (must be image uploads, not video).

Action: ask user to swap in better photos, retrain.

## `Session expired`

`higgsfield auth login`.

## Slow training

Default timeout is 30m. If still in progress: `higgsfield soul-id wait <id> --timeout 60m`.


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
