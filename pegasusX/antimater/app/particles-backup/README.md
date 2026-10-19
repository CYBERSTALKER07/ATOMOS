# Pegasus Particle Systems Backup & Reference

This directory contains standalone copies of all custom particle systems developed for Pegasus OS, preserved so they can be referenced, restored, or reused anywhere in the codebase.

---

## 1. `NeuralParticleCanvas.tsx`
**Location**: [`app/components/NeuralParticleCanvas.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/components/NeuralParticleCanvas.tsx)  
**Backup**: [`app/particles-backup/NeuralParticleCanvas.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/particles-backup/NeuralParticleCanvas.tsx)

### Capabilities:
- **Authentic 4x4 Bayer Dither Matrix**: 10,637 structured points arranged on a regular 3.2px grid representing the face and pointing hand.
- **Dynamic Dither Shimmer**: Sub-pixel harmonic oscillation (`Math.sin(now * 0.0032 + p.idlePhase)`) causing dither dots to subtly pulse and live-dither in place.
- **Starter Dither Sweep**: Assembles dots sequentially from left to right across the face over 1.3s with cubic easing.
- **Interactive Magnetic Physics**: Smooth cursor repulsion with spring return (`force * 5.2`, `springRate = 0.08`, `friction = 0.88`).
- **Click Shockwaves**: Propagates expanding radial impulse ripples across the dither grid.
- **Performance**: High-speed bucketed batch fills (`ctx.fillRect`), zero memory allocations inside RAF render loop, running at 120 FPS.

---

## 2. `neuralFaceData.ts`
**Location**: [`app/data/neuralFaceData.ts`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/data/neuralFaceData.ts)  
**Backup**: [`app/particles-backup/neuralFaceData.ts`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/particles-backup/neuralFaceData.ts)

### Data Architecture:
- Encodes 10,637 points into a compact **7-byte binary stride** base64 string:
  - `uint16 (2 bytes)`: `nx * 10000` (normalized X)
  - `uint16 (2 bytes)`: `ny * 10000` (normalized Y)
  - `uint8 (1 byte)`: `r` (red channel)
  - `uint8 (1 byte)`: `g` (green channel)
  - `uint8 (1 byte)`: `b` (blue channel)
- Decodes instantly in browser via `DataView` into typed `NeuralPoint[]`.

---

## 3. `ParticleText.tsx`
**Location**: [`app/components/ParticleText.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/components/ParticleText.tsx)  
**Backup**: [`app/particles-backup/ParticleText.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/particles-backup/ParticleText.tsx)

### Capabilities:
- Renders any arbitrary text or multi-line headline in interactive particles.
- **`fitWidth` prop**: Automatically calculates container width and scales typography to span full width (used for the footer wordmark).
- **Triggers**: `view`, `mount`, `hover`, `click`, `none`.
- **Spring Physics**: Cursor repel radius, idle drift, customizable gather duration, scatter dispersion, and density.

---

## 4. `EarlyAccessSection.tsx`
**Location**: [`app/components/EarlyAccessSection.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/components/EarlyAccessSection.tsx)  
**Backup**: [`app/particles-backup/EarlyAccessSection.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/particles-backup/EarlyAccessSection.tsx)

### Composition:
- Mounts `NeuralParticleCanvas` in full bleed behind content.
- Headline rendered via `ParticleText` with Markdown styling (`*Access.*`).
- High-contrast pure black `#000000` aesthetic without vignette shadows or blur halos.
- Glassy transparent CTA button (`bg-white/10 hover:bg-white/20`).

---

## 5. `ParticleWorldMap.tsx`
**Location**: [`app/components/ParticleWorldMap.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/components/ParticleWorldMap.tsx)  
**Backup**: [`app/particles-backup/ParticleWorldMap.tsx`](file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/antimater/app/particles-backup/ParticleWorldMap.tsx)

- Global network grid represented in vector particle points.
- Features dynamic pinging nodes, telemetry links, and interactive hover tooltips.
