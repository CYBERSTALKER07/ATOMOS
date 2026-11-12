# Surface Skill: iOS Native Fleet

Governs native iOS development across all 5 iOS apps.
Enforce:
- Swift 6, SwiftUI, Swift Concurrency (Actors, `@MainActor`).
- `NetworkLifecycleEngine.swift` with offline transaction queueing.
- Zero shared cross-app UI packages (local `Theme/`, `Design/`, `Views/`).
- Apple HIG compliance with fluid 120Hz ProMotion spring animations.
