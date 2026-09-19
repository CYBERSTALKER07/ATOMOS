# Navigation and presentation

- Use `NavigationStack` or `NavigationSplitView` as appropriate; flag all use of the deprecated `NavigationView`.
- Strongly prefer to use `navigationDestination(for:)` to specify destinations; flag all use of the old `NavigationLink(destination:)` pattern where it should be replaced.
- Never mix `navigationDestination(for:)` and `NavigationLink(destination:)` in the same navigation hierarchy; it causes significant problems.
- `navigationDestination(for:)` must be registered once per data type; flag duplicates.


## Alerts, confirmation dialogs, and sheets

- Always attach `confirmationDialog()` to the user interface that triggers the dialog. This allows Liquid Glass animations to move from the correct source.
- If an alert has only a single “OK” button that does nothing but dismiss the alert, it can be omitted entirely: `.alert("Dismiss Me", isPresented: $isShowingAlert) { }`.
- If a sheet is designed to present an optional piece of data, prefer `sheet(item:)` over `sheet(isPresented:)` so the optional is safely unwrapped.
- When using `sheet(item:)` with a view that accepts the item as its only initializer parameter, prefer `sheet(item: $someItem, content: SomeView.init)` over `sheet(item: $someItem) { someItem in SomeView(item: someItem) }`.


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
