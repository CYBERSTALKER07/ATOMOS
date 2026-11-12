# Surface Skill: Cloud Infrastructure & SRE

Governs Terraform modules and Kubernetes (GKE) overlays.
Enforce:
- Strict multi-region cell isolation (`cell-uz`, `cell-eu`, `cell-kz`).
- Immutable container digests in production overlays (zero `:latest`).
- Dedicated service separation (`optimizer-core` on its own image).
- Active monitoring with all 6 platform SLO alerts.
