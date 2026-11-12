import SwiftUI

struct ScoredExceptionsView: View {
    @State private var vm = ScoredExceptionsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.rows.isEmpty {
                ProgressView()
            } else if let errorText = vm.errorText, vm.rows.isEmpty {
                Text(errorText).padding()
            } else if vm.rows.isEmpty {
                Text("mobile_supplier.chrome.no_scored_exceptions")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(vm.rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.type.isEmpty ? "—" : row.type)
                            .font(.headline)
                        Text(L10n.format("mobile_supplier.chrome.exception_score", "\(row.score)", row.severity, "\(row.ageMinutes)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.orderId.isEmpty {
                            Text(row.orderId)
                                .font(.caption)
                                .foregroundStyle(.tint)
                        }
                        if !row.topPlaybookName.isEmpty {
                            Text(row.topPlaybookName)
                                .font(.caption)
                        }
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(row.type), score \(row.score), severity \(row.severity)")
                }
            }
        }
        .navigationTitle("mobile_supplier.chrome.control_tower")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
