> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


---
name: ffmpeg
description: Using FFmpeg and FFprobe in Remotion
metadata:
  tags: ffmpeg, ffprobe, video, trimming
---

## FFmpeg in Remotion

`ffmpeg` and `ffprobe` do not need to be installed. They are available via the `npx remotion ffmpeg` and `npx remotion ffprobe`:

```bash
npx remotion ffmpeg -i input.mp4 output.mp3
npx remotion ffprobe input.mp4
```

### Trimming videos

You have 2 options for trimming videos:

1. **Preferred**: Use the `trimBefore` and `trimAfter` props of the `<Video>` component. This is non-destructive, requires no re-encoding, and you can change the trim at any time.

```tsx
import {Video} from '@remotion/media';

<Video src={staticFile('video.mp4')} trimBefore={5 * fps} trimAfter={10 * fps} />;
```

2. Use the FFmpeg command line. You MUST re-encode the video to avoid frozen frames at the start of the video. Only use this if you need a standalone trimmed file (e.g. for upload or external use).

```bash
# Re-encodes from the exact frame
npx remotion ffmpeg -ss 00:00:05 -i public/input.mp4 -to 00:00:10 -c:v libx264 -c:a aac public/output.mp4
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
