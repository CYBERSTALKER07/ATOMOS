import SwiftUI

struct ManifestsView: View {
    @State private var vm = ManifestsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.manifests.isEmpty {
                ContentUnavailableView(
                    "No Manifests",
                    systemImage: "doc.on.doc",
                    description: Text("mobile_warehouse.ui.no_manifests_found")
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.manifests) { manifest in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(String(manifest.manifestId.prefix(8)))
                                    .font(.headline.monospaced())
                                Text(L10n.format("mobile_warehouse.ui.stopcount_stops_drivername", "\(manifest.stopCount)", "\(manifest.driverName)"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                        }
                        .frame(minHeight: 44)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Manifest \(String(manifest.manifestId.prefix(8))), \(manifest.stopCount) stops, driver \(manifest.driverName)")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.manifests")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh manifests")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
