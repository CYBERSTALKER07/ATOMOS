# Marketing asset delivery checklist

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Drop Phase 2 media into this tree. Phase 1 uses tech-icon SVG placeholders via `AssetSlot` — swap is one prop change per section.

## Directory layout

```
public/
├── models/
│   object-a-hero.glb          # Hero — network / control-plane sculpture
│   object-b-layers.glb        # Control plane stack (6 named meshes optional)
│   object-c-roles.glb         # Six-role carousel prop or modular scene
├── video/
│   hero-loop.mp4              # Optional hero background loop
│   cta-horizon.mp4            # Optional CTA background
├── images/
│   architecture-diagram.svg   # Pegasus architecture overview
│   role-supplier.png          # Optional per-role illustrations
└── brand/
    pegasus-logo.svg           # Wordmark + mark (white + black versions)
```

## Slot mapping

| Slot | Section | Component | Constant |
|------|---------|-----------|----------|
| Object A | Hero | `HeroSection` | `ASSET_SLOTS.hero` |
| Object B | Control plane | `ControlPlaneSection` | `ASSET_SLOTS.controlPlane` |
| Object C | Six roles | `RolesParadeSection` | `ASSET_SLOTS.roles` |
| Hero video | Hero (optional) | `AssetSlot` | `ASSET_SLOTS.heroVideo` |
| CTA video | Contact (optional) | `CtaSection` | `ASSET_SLOTS.ctaVideo` |

## 3D specs

- Low-poly preferred (<50k tris per scene)
- Y-up, centered origin
- Single material (we tint white in shader for B&W pass)
- Draco compression OK
- No textures required for Phase 1 monochrome

## Video specs

- H.264, `#000000` background, no audio
- Hero loop: 1920×1080, 5–10s, under 5MB
- Lazy-loaded with reduced-motion static poster fallback

## Swap procedure

1. Place files at paths above (or update `ASSET_SLOTS` in `lib/constants.ts`)
2. Set `AssetSlot` `src` prop or enable glTF loader in section component
3. Remove tech-icon fallback when asset loads successfully

Legacy R3F primitives remain in `components/three/` for reference during Phase 2 wiring.
