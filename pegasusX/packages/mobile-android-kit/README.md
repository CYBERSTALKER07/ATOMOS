# mobile-android-kit

Shared Android field-layer kit (§8.8): offline queue contract, HTTP flush semantics,
reconnect backoff, client-policy snapshot.

Apps include via per-project `settings.gradle.kts`:

```kotlin
include(":mobile-kit")
project(":mobile-kit").projectDir = file("../../packages/mobile-android-kit")
```

Version catalog (optional): [`../../gradle/libs.versions.toml`](../../gradle/libs.versions.toml).
