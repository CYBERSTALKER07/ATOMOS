import SwiftUI

struct PlaybooksView: View {
    @State private var vm = PlaybooksViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.rows.isEmpty {
                ProgressView()
            } else if let errorText = vm.errorText, vm.rows.isEmpty {
                Text(errorText).padding()
            } else if vm.rows.isEmpty {
                Text("mobile_supplier.chrome.no_playbooks")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(vm.rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.name.isEmpty ? row.playbookId : row.name)
                            .font(.headline)
                        Text(L10n.format("mobile_supplier.chrome.playbook_meta", row.isActive ? L10n.format("mobile_supplier.chrome.state_active") : L10n.format("mobile_supplier.chrome.state_inactive"), "\(row.priority)", row.autoExecute ? L10n.format("mobile_supplier.chrome.auto_suffix") : ""))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.description.isEmpty {
                            Text(row.description).font(.caption)
                        }
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(row.name.isEmpty ? row.playbookId : row.name), \(row.isActive ? "Active" : "Inactive"), priority \(row.priority)")
                }
            }
        }
        .navigationTitle("mobile_supplier.chrome.playbooks")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
