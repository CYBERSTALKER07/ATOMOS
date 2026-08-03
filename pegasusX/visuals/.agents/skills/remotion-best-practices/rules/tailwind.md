> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


---
name: tailwind
description: Using TailwindCSS in Remotion.
metadata:
---

You can and should use TailwindCSS in Remotion, if TailwindCSS is installed in the project.

Don't use `transition-*` or `animate-*` classes - always animate using the `useCurrentFrame()` hook.

Tailwind must be installed and enabled first in a Remotion project - fetch https://www.remotion.dev/docs/tailwind using WebFetch for instructions.
