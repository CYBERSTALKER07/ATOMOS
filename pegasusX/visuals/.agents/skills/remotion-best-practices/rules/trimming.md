> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


---
name: trimming
description: Trimming patterns for Remotion - cut the beginning or end of animations
metadata:
  tags: sequence, trim, clip, cut, offset
---

Use `<Sequence>` with a negative `from` value to trim the start of an animation.

## Trim the Beginning

A negative `from` value shifts time backwards, making the animation start partway through:

```tsx
import { Sequence, useVideoConfig } from "remotion";

const fps = useVideoConfig();

<Sequence from={-0.5 * fps}>
  <MyAnimation />
</Sequence>;
```

The animation appears 15 frames into its progress - the first 15 frames are trimmed off.
Inside `<MyAnimation>`, `useCurrentFrame()` starts at 15 instead of 0.

## Trim the End

Use `durationInFrames` to unmount content after a specified duration:

```tsx
<Sequence durationInFrames={1.5 * fps}>
  <MyAnimation />
</Sequence>
```

The animation plays for 45 frames, then the component unmounts.

## Trim and Delay

Nest sequences to both trim the beginning and delay when it appears:

```tsx
<Sequence from={30}>
  <Sequence from={-15}>
    <MyAnimation />
  </Sequence>
</Sequence>
```

The inner sequence trims 15 frames from the start, and the outer sequence delays the result by 30 frames.


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
