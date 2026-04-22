# Native Mobile Safety — Swift/SwiftUI & Kotlin/Compose Anti-Pattern Prevention

## Description
Prevents crashes, memory leaks, concurrency bugs, and type drift in native iOS (SwiftUI) and Android (Kotlin/Compose) applications. Activates when writing or reviewing code in driver, retailer, factory, or warehouse native apps.

## Trigger Keywords
swift, swiftui, kotlin, compose, android, ios, viewmodel, coroutine, scope, force unwrap, try!, weak self, SupervisorJob, MainActor, CoroutineScope, Codable, Serializable, URLSession, OkHttp, Retrofit

## Anti-Pattern Catalog — Swift / SwiftUI

### S1. Force-Unwrap URL Construction (CRASH)
```swift
// ❌ WRONG — crashes if apiBaseURL contains spaces or invalid chars
let url = URL(string: "\(APIClient.shared.apiBaseURL)/v1/sync/batch")!

// ✅ RIGHT — guard let with graceful failure
guard let url = URL(string: "\(APIClient.shared.apiBaseURL)/v1/sync/batch") else {
    slog.error("invalid URL for sync batch")
    return
}
```
**Real findings**: `SyncServiceLive.swift` L33, `TelemetryViewModel.swift` L41 — both force-unwrap URL construction from string interpolation.

**Rule**: `URL(string:)` returns optional. NEVER force-unwrap. Use `guard let` with logging + early return.

### S2. try! in Production Paths (UNRECOVERABLE CRASH)
```swift
// ❌ WRONG — SwiftData migration failure = app crash
let container = try! ModelContainer(for: OfflineDelivery.self, configurations: config)

// ✅ RIGHT — do/catch with user-facing recovery
do {
    let container = try ModelContainer(for: OfflineDelivery.self, configurations: config)
    self.modelContainer = container
} catch {
    logger.error("SwiftData init failed: \(error)")
    // Show error UI, offer to clear local data, etc.
    self.modelContainer = nil
    self.showDataCorruptionAlert = true
}
```
**Real finding**: `OfflineVerifierView.swift` L303 — `try!` on SwiftData initialization.

**Rule**: `try!` is acceptable ONLY in:
- Unit tests
- SwiftUI previews
- Compile-time-guaranteed operations (e.g., `try! JSONEncoder().encode(staticValue)`)

Never in production code paths that involve I/O, file system, or data migration.

### S3. Missing [weak self] in Escaping Closures (RETAIN CYCLE)
```swift
// ❌ WRONG — Task captures self strongly in a class-based ViewModel
class FleetViewModel: ObservableObject {
    func startPolling() {
        Task {
            while !Task.isCancelled {
                await self.fetchFleet() // strong capture → ViewModel never deallocates
                try? await Task.sleep(for: .seconds(5))
            }
        }
    }
}

// ✅ RIGHT — weak self with guard
class FleetViewModel: ObservableObject {
    func startPolling() {
        Task { [weak self] in
            while !Task.isCancelled {
                guard let self else { return }
                await self.fetchFleet()
                try? await Task.sleep(for: .seconds(5))
            }
        }
    }
}
```
**When `[weak self]` is needed**: class-based ViewModels with long-lived Tasks, completion handlers, NotificationCenter observers, Timer callbacks.

**When NOT needed**: `@MainActor`-annotated classes where the Task is scoped to view lifetime, struct-based SwiftUI views (value types can't retain-cycle).

### S4. @MainActor for ViewModels
```swift
// ❌ WRONG — UI updates from background thread
class OrderViewModel: ObservableObject {
    @Published var orders: [Order] = []

    func fetchOrders() {
        Task {
            let result = await api.getOrders()
            orders = result // ⚠️ may crash — @Published write from background
        }
    }
}

// ✅ RIGHT — ViewModel is @MainActor
@MainActor
class OrderViewModel: ObservableObject {
    @Published var orders: [Order] = []

    func fetchOrders() {
        Task {
            let result = await api.getOrders() // runs on MainActor
            orders = result // safe — MainActor
        }
    }
}
```

### S5. Codable Models Must Link to Backend
```swift
// ✅ Every Codable struct mirrors a backend Go type — comment the link
/// Mirror of backend-go/models.Driver — keep JSON keys aligned
struct Driver: Codable {
    let driverId: String      // json:"driver_id"
    let name: String           // json:"name"
    let supplierId: String     // json:"supplier_id"
    let homeNodeType: String   // json:"home_node_type" — Phase VII
    let homeNodeId: String     // json:"home_node_id"   — Phase VII

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name
        case supplierId = "supplier_id"
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
    }
}
```

---

## Anti-Pattern Catalog — Kotlin / Compose

### K1. CoroutineScope Without SupervisorJob (CASCADING FAILURE)
```kotlin
// ❌ WRONG — child failure cancels ALL siblings
val scope = CoroutineScope(Dispatchers.IO)
scope.launch { uploadTelemetry() }   // if this fails...
scope.launch { maintainWebSocket() } // ...this gets cancelled too

// ✅ RIGHT — SupervisorJob isolates child failures
val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
scope.launch { uploadTelemetry() }   // failure is isolated
scope.launch { maintainWebSocket() } // continues running
```
**Real finding**: `TelemetrySocket.kt` L39 — `CoroutineScope(Dispatchers.IO)` without `SupervisorJob()`.

### K2. Custom Scope Without cancel() (MEMORY LEAK)
```kotlin
// ❌ WRONG — scope leaks coroutines
class TelemetrySocket {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    // No cancel() anywhere — coroutines run forever
}

// ✅ RIGHT — explicit lifecycle management
class TelemetrySocket : Closeable {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun close() {
        scope.cancel() // cancels all child coroutines
    }
}
// Called from Service.onDestroy() or ViewModel.onCleared()
```
**Rule**: Every manually-created `CoroutineScope` MUST have a matching `scope.cancel()` call in the teardown path. Prefer `viewModelScope` (auto-cancelled on `onCleared()`) or `lifecycleScope` (auto-cancelled on DESTROYED).

### K3. Network Call on Main Thread (ANR)
```kotlin
// ❌ WRONG — blocks main thread
fun fetchOrders() {
    val response = client.newCall(request).execute() // BLOCKS MAIN
    _orders.value = parseResponse(response)
}

// ✅ RIGHT — IO dispatcher
fun fetchOrders() {
    viewModelScope.launch {
        val response = withContext(Dispatchers.IO) {
            client.newCall(request).execute()
        }
        _orders.value = parseResponse(response)
    }
}
```
**Exception**: OkHttp interceptors run on the OkHttp dispatcher thread (not Main). `client.newCall(refreshRequest).execute()` inside an interceptor is the canonical token-refresh pattern.

### K4. GlobalScope Is Banned
```kotlin
// ❌ WRONG — unstructured, no lifecycle management, no cancellation
GlobalScope.launch { uploadAnalytics() }

// ✅ RIGHT — scoped to a lifecycle owner
viewModelScope.launch { uploadAnalytics() }
// or: custom scope with explicit cancel()
```

### K5. @Serializable Models Must Link to Backend
```kotlin
// ✅ Every @Serializable class mirrors a backend Go type
/** Mirror of backend-go/models.Driver — keep JSON keys aligned */
@Serializable
data class Driver(
    @SerialName("driver_id") val driverId: String,
    @SerialName("name") val name: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("home_node_type") val homeNodeType: String, // Phase VII
    @SerialName("home_node_id") val homeNodeId: String,      // Phase VII
)
```

---

## Type Drift Prevention (Both Platforms)

When the backend Go struct adds a field:
1. Update `packages/types` TypeScript interface
2. Update Swift `Codable` struct (all iOS apps)
3. Update Kotlin `@Serializable` class (all Android apps)
4. Verify JSON key (`snake_case`) matches `json:"..."` Go tag

**Real finding**: `packages/types/entities.ts` `Driver` and `Vehicle` interfaces are missing `HomeNodeType`, `HomeNodeId` — fields written by all four fleet-creation handlers.

## Verification Checklist — Swift
- [ ] Zero `!` force-unwraps on `URL(string:)`, `JSONDecoder.decode`, or file operations
- [ ] Zero `try!` in production paths (tests/previews only)
- [ ] `[weak self]` in escaping closures on class-based types
- [ ] `@MainActor` on ObservableObject ViewModels
- [ ] Every Codable struct has a comment linking to the backend canonical type
- [ ] CodingKeys match backend `json:"..."` tags exactly

## Verification Checklist — Kotlin
- [ ] Every custom `CoroutineScope` includes `SupervisorJob()`
- [ ] Every custom scope has a matching `scope.cancel()` in teardown
- [ ] No `GlobalScope.launch` — use `viewModelScope` or lifecycle-scoped
- [ ] Network calls use `Dispatchers.IO`, never `Dispatchers.Main`
- [ ] Every `@Serializable` class has a comment linking to the backend type
- [ ] `@SerialName` values match backend `json:"..."` tags exactly

## Cross-References
- **intrusions.md** §10 (Native Mobile Traps) — full finding details
- **intrusions.md** §11 (Schema & Type Drift) — cross-platform alignment
- **gemini-instructions.md** Cross-Role Synchronization Doctrine — all apps per role ship together
- **swiftui-pro** skill — SwiftUI code review best practices
- **m3** skill — Material Design 3 for Android/Compose
