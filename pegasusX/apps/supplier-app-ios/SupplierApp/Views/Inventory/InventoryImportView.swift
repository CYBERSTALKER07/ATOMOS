import SwiftUI

private let sampleCSV = """
product_id,warehouse_id,quantity_on_hand,reorder_threshold
sku-apple,wh-main,120,20
sku-pear,wh-main,80,15
"""

struct InventoryImportView: View {
    @State private var csvText = sampleCSV
    @State private var fileName = "inventory.csv"
    @State private var step = 0
    @State private var sessionId: String?
    @State private var sessionStatus = ""
    @State private var mappings: [SupplierImportMappingCandidate] = []
    @State private var applyResult: SupplierImportApplyResponse?
    @State private var directResult: SupplierInventoryImportResult?
    @State private var busy = false
    @State private var error: String?

    private let stepLabels = ["Upload", "Mapping", "Review", "Done"]

    var body: some View {
        Form {
            Section {
                Picker("Step", selection: $step) {
                    ForEach(stepLabels.indices, id: \.self) { index in
                        Text(stepLabels[index]).tag(index)
                    }
                }
                .pickerStyle(.segmented)
            }

            if step == 0 {
                Section("CSV payload") {
                    TextField("mobile_supplier.ui.file_name", text: $fileName)
                    TextEditor(text: $csvText)
                        .frame(minHeight: 160)
                        .font(.caption.monospaced())
                }
                Section {
                    Button(busy ? "Importing…" : "Direct CSV import") {
                        Task { await directImport() }
                    }
                    .disabled(busy || csvText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

                    Button(busy ? "Starting…" : "Start wizard import") {
                        Task { await startWizard() }
                    }
                    .disabled(busy || csvText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }

            if step == 1 {
                Section("Column mapping") {
                    if mappings.isEmpty {
                        Text("mobile_supplier.ui.no_mappings_suggested_yet")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(mappings) { mapping in
                            VStack(alignment: .leading) {
                                Text(mapping.sourceColumn).font(.headline)
                                Text(L10n.format("mobile_supplier.ui.targetfield_confidence_100", "\(mapping.targetField)", "\(Int(mapping.confidence * 100))"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    if sessionId != nil {
                        Button("mobile_supplier.ui.continue_to_review") { step = 2 }
                    }
                }
            }

            if step == 2, let sessionId {
                Section(L10n.format("mobile_supplier.ui.review_session_prefix", "\(sessionId.prefix(8))")) {
                    Text(L10n.format("mobile_supplier.ui.status_sessionstatus_2", "\(sessionStatus)"))
                    Button(busy ? "Approving…" : "Approve & apply") {
                        Task { await approveAndApply(sessionId) }
                    }
                    .disabled(busy)
                }
            }

            if step == 3 {
                Section("Result") {
                    if let applyResult {
                        Text(L10n.format("mobile_supplier.ui.applied_appliedrows_rows_status", "\(applyResult.appliedRows)", "\(applyResult.status)"))
                    } else if let directResult {
                        Text(L10n.format("mobile_supplier.ui.direct_import_applied_applied_skipped_skipped", "\(directResult.applied)", "\(directResult.skipped)"))
                    } else {
                        Text("mobile_supplier.ui.import_complete")
                    }
                }
            }

            if let error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
        }
        .navigationTitle("mobile_supplier.ui.import_inventory")
    }

    private func directImport() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            directResult = try await SupplierService.importInventoryCSV(
                csvText,
                idempotencyKey: UUID().uuidString
            )
            step = 3
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func startWizard() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let scopeId = SupplierIdempotencyKeys.supplierScopeId()
            let bytes = csvText.utf8.count
            let created = try await SupplierOperationsService.createImportSession(
                fileName: fileName,
                fileSizeBytes: bytes,
                idempotencyKey: SupplierIdempotencyKeys.importCreate(scopeId: scopeId, fileName: fileName, fileSizeBytes: bytes)
            )
            let ingested = try await SupplierOperationsService.ingestImportSession(
                sessionId: created.sessionId,
                csv: csvText,
                idempotencyKey: SupplierIdempotencyKeys.importIngest(sessionId: created.sessionId, csvBody: csvText)
            )
            let mapping = try await SupplierOperationsService.getImportMapping(sessionId: created.sessionId)
            sessionId = created.sessionId
            sessionStatus = ingested.status
            mappings = mapping.mappingJson?.mappings ?? []
            step = 1
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func approveAndApply(_ sessionId: String) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            try await SupplierOperationsService.approveImportSession(
                sessionId: sessionId,
                idempotencyKey: SupplierIdempotencyKeys.importApprove(sessionId: sessionId)
            )
            applyResult = try await SupplierOperationsService.applyImportSession(
                sessionId: sessionId,
                idempotencyKey: SupplierIdempotencyKeys.importApply(sessionId: sessionId)
            )
            step = 3
        } catch {
            self.error = error.localizedDescription
        }
    }
}
