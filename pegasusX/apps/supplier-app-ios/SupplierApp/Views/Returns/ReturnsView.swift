import SwiftUI

struct ReturnsView: View {
    @State private var items: [SupplierReturnRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var resolvingId: String?
    @State private var resolution = "RETURN_TO_STOCK"
    @State private var notes = ""
    @State private var actionLoading: String?

    private let resolutions = ["RETURN_TO_STOCK", "WRITE_OFF"]

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading returns…")
            } else if let error, items.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if items.isEmpty {
                SupplierEmptyView(
                    title: "No open returns",
                    message: "Rejected delivery quantities appear here after driver offload."
                )
            } else {
                List(items) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.productName).font(.headline)
                        Text("Qty \(row.quantity) · \(row.reason)")
                            .font(.subheadline)
                        Text("Physical: \(row.physicalStatus)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.driverName.isEmpty {
                            Text("Driver: \(row.driverName)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if row.receivedQty > 0 {
                            Text("Scanned: \(row.receivedQty)")
                                .font(.caption2)
                                .foregroundStyle(.orange)
                        }

                        if resolvingId == row.returnId {
                            Picker("Resolution", selection: $resolution) {
                                ForEach(resolutions, id: \.self) { option in
                                    Text(option.replacingOccurrences(of: "_", with: " ")).tag(option)
                                }
                            }
                            .pickerStyle(.menu)
                            TextField("Notes (optional)", text: $notes)
                                .textInputAutocapitalization(.sentences)
                            HStack {
                                Button(actionLoading == row.returnId ? "…" : "Confirm") {
                                    Task { await resolve(returnId: row.returnId) }
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(actionLoading == row.returnId)
                                Button("Cancel") { resolvingId = nil }
                                    .buttonStyle(.bordered)
                            }
                        } else if row.physicalStatus == "RESTOCKED" || row.physicalStatus == "WRITTEN_OFF" {
                            Text("Gate resolved")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        } else {
                            Button("Dispute / override") {
                                resolvingId = row.returnId
                                resolution = "RETURN_TO_STOCK"
                                notes = ""
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                    .padding(.vertical, 4)
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Returns")
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.returns(status: "PENDING", limit: 100, offset: 0)
            items = resp.data
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func resolve(returnId: String) async {
        actionLoading = returnId
        defer { actionLoading = nil }
        do {
            let key = SupplierIdempotency.resolveReturn(returnId: returnId, resolution: resolution)
            try await SupplierOperationsService.resolveReturn(
                returnId: returnId,
                resolution: resolution,
                notes: notes,
                idempotencyKey: key
            )
            resolvingId = nil
            notes = ""
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
