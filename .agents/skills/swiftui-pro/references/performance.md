# Performance

- When toggling modifier values, prefer ternary expressions over if/else view branching to avoid `_ConditionalContent`, preserve structural identity, and avoid repeatedly recreating underlying platform views.
- Avoid `AnyView` unless absolutely required. Use `@ViewBuilder`, `Group`, or generics instead.
- If a `ScrollView` has an opaque, static, and solid background, prefer to use `scrollContentBackground(.visible)` to improve scroll-edge rendering efficiency.
- It is more efficient to break views up by making dedicated SwiftUI views rather than place them into computed properties or methods. Using `@ViewBuilder` on a property or method does not solve this; breaking views up is strongly preferred.
- Always ensure view initializers are kept as small and simple as possible, avoiding any non-trivial work. Flag any work that can be moved into a `task()` modifier to be run when the view is shown.
- Similarly, assume each view’s `body` property is called frequently – if logic such as sorting or filtering can be moved out of there easily, it should be.
- Avoid creating properties to store formatters such as `DateFormatter` unless they are required. A more natural approach is to use `Text` with a format, like this: `Text(Date.now, format: .dateTime.day().month().year())` or `Text(100, format: .currency(code: "USD"))`.
- Avoid expensive inline transforms in `List`/`ForEach` initializers (e.g. `items.filter { ... }`) when they are repeated often.
- Prefer deriving transformed data from the source-of-truth using `let`, or caching in `@State`. However, do not cache derived collections in `@State` unless you also own explicit invalidation logic to avoid stale UI.
- For large data sets in `ScrollView`, use `LazyVStack`/`LazyHStack`; flag eager stacks with many children.
- Prefer using `task()` over `onAppear()` when doing async work, because it will be cancelled automatically when the view disappears.
- Avoid storing escaping `@ViewBuilder` closures on views when possible; store built view results instead.

Example:

```swift
// Anti-pattern: stores an escaping closure on the view.
struct CardView<Content: View>: View {
    let content: () -> Content

    var body: some View {
        VStack(alignment: .leading) {
            content()
        }
        .padding()
        .background(.ultraThinMaterial)
        .clipShape(.rect(cornerRadius: 8))
    }
}

// Preferred: store the built view value; the synthesized init handles calling the builder.
struct CardView<Content: View>: View {
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading) {
            content
        }
        .padding()
        .background(.ultraThinMaterial)
        .clipShape(.rect(cornerRadius: 8))
    }
}
```


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
