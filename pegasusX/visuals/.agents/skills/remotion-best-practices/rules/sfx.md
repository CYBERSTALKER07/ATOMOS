> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


---
name: sfx
description: Including sound effects
metadata:
  tags: sfx, sound, effect, audio
---

To include a sound effect, use the `<Audio>` tag:

```tsx
import { Audio } from "@remotion/sfx";

<Audio src={"https://remotion.media/whoosh.wav"} />;
```

The following sound effects are available:

- `https://remotion.media/whoosh.wav`
- `https://remotion.media/whip.wav`
- `https://remotion.media/page-turn.wav`
- `https://remotion.media/switch.wav`
- `https://remotion.media/mouse-click.wav`
- `https://remotion.media/shutter-modern.wav`
- `https://remotion.media/shutter-old.wav`
- `https://remotion.media/ding.wav`
- `https://remotion.media/bruh.wav`
- `https://remotion.media/vine-boom.wav`
- `https://remotion.media/windows-xp-error.wav`
- `https://remotion.media/fah.wav`
- `https://remotion.media/spongebob-fail.wav`
- `https://remotion.media/omg-hell-nah.wav`
- `https://remotion.media/price-is-right-fail.wav`
- `https://remotion.media/romance-meme.wav`
- `https://remotion.media/bone-crack.wav`
- `https://remotion.media/anime-wow.wav`
- `https://remotion.media/yippee.wav`
- `https://remotion.media/loading-lag.wav`
- `https://remotion.media/wilhelm-scream.wav`
- `https://remotion.media/mac-quack.wav`
- `https://remotion.media/skedaddle.wav`
- `https://remotion.media/snapchat-notification.wav`
- `https://remotion.media/nelly-ahh.wav`
- `https://remotion.media/sanctuary-guardian-what.wav`
- `https://remotion.media/minecraft-hurt.wav`
- `https://remotion.media/oh-my-god-vine.wav`
- `https://remotion.media/illuminati-confirmed.wav`
- `https://remotion.media/dramatic-boomer.wav`
- `https://remotion.media/triggered.wav`
- `https://remotion.media/record-scratch.wav`

For more sound effects, search the internet. A good resource is https://github.com/kapishdima/soundcn/tree/main/assets.


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
