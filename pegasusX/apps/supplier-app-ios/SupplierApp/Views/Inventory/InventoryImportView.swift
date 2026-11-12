import SwiftUI

struct InventoryImportView: View {
    @State private var viewModel = InventoryImportViewModel()

    private let stepLabels = ["mobile_supplier.chrome.upload", "mobile_supplier.chrome.mapping", "mobile_supplier.chrome.review", "mobile_supplier.chrome.done"]

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            Section {
                Picker("mobile_supplier.chrome.step", selection: $vm.step) {
                    ForEach(stepLabels.indices, id: \.self) { index in
                        Text(LocalizedStringKey(stepLabels[index])).tag(index)
                    }
                }
                .pickerStyle(.segmented)
            }

            if viewModel.step == 0 {
                Section("mobile_supplier.chrome.csv_payload") {
                    TextField("mobile_supplier.ui.file_name", text: $vm.fileName)
                    TextEditor(text: $vm.csvText)
                        .frame(minHeight: 160)
                        .font(.caption.monospaced())
                }
                Section {
                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.importing" : "mobile_supplier.chrome.direct_csv_import")) {
                        Task { await viewModel.directImport() }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy || viewModel.csvText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.starting" : "mobile_supplier.chrome.start_wizard_import")) {
                        Task { await viewModel.startWizard() }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy || viewModel.csvText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }

            if viewModel.step == 1 {
                Section("mobile_supplier.chrome.column_mapping") {
                    if viewModel.mappings.isEmpty {
                        Text("mobile_supplier.ui.no_mappings_suggested_yet")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(viewModel.mappings) { mapping in
                            VStack(alignment: .leading) {
                                Text(mapping.sourceColumn).font(.headline)
                                Text(L10n.format("mobile_supplier.ui.targetfield_confidence_100", "\(mapping.targetField)", "\(Int(mapping.confidence * 100))"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                        }
                    }
                    if viewModel.sessionId != nil {
                        Button("mobile_supplier.ui.continue_to_review") { viewModel.step = 2 }
                            .frame(minHeight: 44)
                    }
                }
            }

            if viewModel.step == 2, let sessionId = viewModel.sessionId {
                Section(L10n.format("mobile_supplier.ui.review_session_prefix", "\(sessionId.prefix(8))")) {
                    Text(L10n.format("mobile_supplier.ui.status_sessionstatus_2", "\(viewModel.sessionStatus)"))
                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.approving" : "mobile_supplier.chrome.approve_and_apply")) {
                        Task { await viewModel.approveAndApply(sessionId) }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy)
                }
            }

            if viewModel.step == 3 {
                Section("mobile_supplier.chrome.result") {
                    if let applyResult = viewModel.applyResult {
                        Text(L10n.format("mobile_supplier.ui.applied_appliedrows_rows_status", "\(applyResult.appliedRows)", "\(applyResult.status)"))
                    } else if let directResult = viewModel.directResult {
                        Text(L10n.format("mobile_supplier.ui.direct_import_applied_applied_skipped_skipped", "\(directResult.applied)", "\(directResult.skipped)"))
                    } else {
                        Text("mobile_supplier.ui.import_complete")
                    }
                }
            }

            if let error = viewModel.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_supplier.ui.import_inventory")
    }
}
