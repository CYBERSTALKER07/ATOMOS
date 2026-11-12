# Surface Skill: Android Native Fleet

Governs native Android development across all 5 Android apps.
Enforce:
- Kotlin 2.0+, Jetpack Compose, Coroutines/Flow, MVI architecture.
- `NetworkLifecycleEngine.kt` for structured concurrency and exponential backoff retry.
- Zero shared UI modules across apps (`UiIndependenceTest`).
- Zero mock data in production builds.
- Hardware barcode scanner integration and 150m/50m doorstep geofencing.
