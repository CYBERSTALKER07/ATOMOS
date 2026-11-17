import SwiftUI

struct ManifestExceptionsView: View {
    @State private var vm = ManifestExceptionsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.exceptions.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.exceptions.isEmpty {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.selection()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            } else if vm.exceptions.isEmpty {
                ContentUnavailableView(
                    vm.escalatedOnly ? "No Escalated Exceptions" : "No Exceptions",
                    systemImage: "checkmark.circle",
                    description: Text(
                        vm.escalatedOnly
                            ? "No transfers have hit the DLQ threshold (3+ overflows)."
                            : "All manifest loading operations completed without exceptions."
                    )
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(Array(vm.exceptions.enumerated()), id: \.element.id) { index, exception in
                        ExceptionRow(
                            exception: exception,
                            resolving: vm.resolvingId == exception.exceptionId,
                            onResolve: {
                                Task { await vm.resolve(exception.exceptionId) }
                            }
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
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    vm.toggleEscalated()
                } label: {
                    Image(systemName: vm.escalatedOnly ? "line.3.horizontal.decrease.circle.fill" : "line.3.horizontal.decrease.circle")
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(vm.escalatedOnly ? "Show all exceptions" : "Filter escalated exceptions only")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel("Refresh gate exceptions")
            }
        }
        .task(id: vm.escalatedOnly) {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}
