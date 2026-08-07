import SwiftUI

struct ManifestExceptionsView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var exceptions: [ManifestException] = []
    @State private var loading = true
    @State private var error: String?
    @State private var escalatedOnly = false
    @State private var resolvingId: String?

    var body: some View {
        Group {
            if loading && exceptions.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error, exceptions.isEmpty {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if exceptions.isEmpty {
                ContentUnavailableView(
                    escalatedOnly ? "No Escalated Exceptions" : "No Exceptions",
                    systemImage: "checkmark.circle",
                    description: Text(
                        escalatedOnly
                            ? "No transfers have hit the DLQ threshold (3+ overflows)."
                            : "All manifest loading operations completed without exceptions."
                    )
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(Array(exceptions.enumerated()), id: \.element.id) { index, exception in
                        ExceptionRow(
                            exception: exception,
                            resolving: resolvingId == exception.exceptionId,
                            onResolve: { resolve(exception.exceptionId) }
                        )
                        .staggeredAppear(index: index)
                    }
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.gate_exceptions")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button("common.action.close", systemImage: "xmark") {
                    dismiss()
                }
                .labelStyle(.iconOnly)
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    escalatedOnly.toggle()
                    load()
                } label: {
                    Image(systemName: escalatedOnly ? "line.3.horizontal.decrease.circle.fill" : "line.3.horizontal.decrease.circle")
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button { load() } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .task(id: escalatedOnly) { load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard let eventType = event.eventType else { return }
                    guard eventType == .manifestUpdate || eventType == .transferUpdate else { return }
                    load(silent: true)
                }
            )
        }
        .onDisappear {
            realtimeClient.disconnect()
        }
    }

    private func load(silent: Bool = false) {
        Task { @MainActor in
            if !silent {
                loading = true
            }
            error = nil
            do {
                let response = try await FactoryService.manifestExceptions(escalatedOnly: escalatedOnly)
                exceptions = response.exceptions
            } catch {
                self.error = error.localizedDescription
            }
            if !silent {
                loading = false
            }
        }
    }

    private func resolve(_ exceptionId: String) {
        Task { @MainActor in
            resolvingId = exceptionId
            do {
                _ = try await FactoryService.resolveManifestException(exceptionId: exceptionId)
                load(silent: true)
            } catch {
                self.error = error.localizedDescription
            }
            resolvingId = nil
        }
    }
}
