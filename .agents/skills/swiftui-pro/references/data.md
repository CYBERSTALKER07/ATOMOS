# Data flow, shared state, and property wrappers

It is important that SwiftUI body code and logic code be kept separate in order to make code easier to read, write, and maintain. That usually means placing code into methods rather than inline in the `body` property, but often also means carving functionality out into separate `@Observable` classes.

These rules help ensure code is efficient and works well in the long term.


## Shared state

- `@Observable` classes must be marked `@MainActor` unless the project has Main Actor default actor isolation. Flag any `@Observable` class missing this annotation.
- All shared data should use `@Observable` classes with `@State` (for ownership) and `@Bindable` / `@Environment` (for passing).
- Strongly prefer not to use `ObservableObject`, `@Published`, `@StateObject`, `@ObservedObject`, or `@EnvironmentObject` unless they are unavoidable, or if they exist in legacy/integration contexts when changing architecture would be complicated.


## Local state

- `@State` should be marked `private` and only owned by the view that created it.
- If a view stores a class instance that contains expensive-to-recompute data, e.g. `CIContext`, it can be stored using `@State` even though it is not an observable object. This effectively uses `@State` as a cache – storing something persistently, but not doing any change tracking on it since it's not an observable object.


## Bindings

- Strongly prefer to avoid creating bindings using `Binding(get:set:)` in view body code. It is much cleaner and simpler to use a binding provided by `@State`, `@Binding` or similar, then use `onChange()` to trigger any effects.
- If the user needs to enter a number into a `TextField`, bind the `TextField` to a numeric value such as `Int` or `Double`, then use its `format` initializer like this: `TextField("Enter your score", value: $score, format: .number)`. Apply either `.keyboardType(.numberPad)` (for integers) or `.keyboardType(.decimalPad)` (for floating-point numbers) as appropriate. Using the modifier alone is *not* sufficient.


## Working with data

- Prefer to make structs conform to `Identifiable` rather than using `id: \.someProperty` in SwiftUI code.
- Never attempt to use `@AppStorage` inside an `@Observable` class, even if marked `@ObservationIgnored` – it will *not* trigger view updates when a change happens.


## SwiftData

- If you only need the number of items matching a query, consider `ModelContext.fetchCount()` with a fetch descriptor. This will *not* live update if the data changes unless something else triggers the update, such as `@Query`, so it should be used carefully.

For more help with SwiftData, suggest the [SwiftData Pro agent skill](https://github.com/twostraws/swiftdata-agent-skill).

## If the project uses SwiftData with CloudKit

- Never use `@Attribute(.unique)`.
- Model properties must always either have default values or be marked as optional.
- All relationships must be marked optional.


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
