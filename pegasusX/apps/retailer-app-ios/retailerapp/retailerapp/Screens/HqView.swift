import SwiftUI

struct HqView: View {
    @State private var vm = HqViewModel()

    var body: some View {
        List {
            if vm.isLoading && !vm.summaryReady && vm.loadError == nil {
                Section {
                    ProgressView("Loading…")
                }
            }
            if let loadError = vm.loadError {
                Section {
                    Text(loadError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                    Button("common.action.retry") {
                        Haptics.light()
                        Task { await vm.load() }
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Retry HQ summary")
                }
            }
            if vm.summaryReady && vm.loadError == nil {
                Section("Summary") {
                    Text(vm.summaryLine)
                        .font(.system(.subheadline, design: .rounded))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("HQ summary: \(vm.summaryLine)")
                }
                Section("Sales by location") {
                    if vm.locations.isEmpty {
                        Text("No HQ rows for this day.")
                            .foregroundStyle(AppTheme.textSecondary)
                    } else {
                        ForEach(vm.locations, id: \.locationId) { loc in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(loc.locationId)
                                    .font(.headline)
                                Text("Qty \(loc.qtySold) · Net \(vm.formatMinor(loc.netMinor))")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            .padding(.vertical, 2)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Location \(loc.locationId), quantity sold \(loc.qtySold), net \(vm.formatMinor(loc.netMinor))")
                        }
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("HQ multi-store")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
