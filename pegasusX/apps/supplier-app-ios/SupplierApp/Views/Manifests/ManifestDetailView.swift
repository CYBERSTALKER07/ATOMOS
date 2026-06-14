import SwiftUI

struct ManifestDetailView: View {
    let manifestId: String
    @State private var detail: SupplierManifestDetail?
    @State private var loading = true
    @State private var error: String?
    @State private var actionError: String?
    @State private var busy = false
    @State private var injectOrderId = ""

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading manifest…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let detail {
                manifestContent(detail)
            } else {
                SupplierEmptyView(title: "Not found", message: "Manifest could not be loaded.")
            }
        }
        .navigationTitle("Manifest")
        .task { await load() }
    }

    @ViewBuilder
    private func manifestContent(_ detail: SupplierManifestDetail) -> some View {
        let state = (detail.state.isEmpty ? detail.status : detail.state).uppercased()
        Form {
            if let actionError {
                Section { Text(actionError).foregroundStyle(.red) }
            }
            Section("Summary") {
                Text(detail.manifestId).font(.caption.monospaced())
                Text("\(detail.status) · \(detail.ordersCount) orders")
                Text(detail.driverName.isEmpty ? (detail.driverId ?? "—") : detail.driverName)
                if let plate = detail.vehiclePlate { Text("Vehicle \(plate)") }
            }
            if state == "DRAFT" {
                Section {
                    Button(busy ? "Starting…" : "Start loading") {
                        Task { await runAction { try await SupplierOperationsService.startManifestLoading(manifestId, idempotencyKey: idem("start-loading")) } }
                    }
                    .disabled(busy)
                }
            }
            if state == "LOADING" {
                Section("Inject order") {
                    TextField("Order ID", text: $injectOrderId)
                    Button("Inject order") {
                        Task {
                            let orderId = injectOrderId.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !orderId.isEmpty else { return }
                            await runAction {
                                try await SupplierOperationsService.injectManifestOrder(
                                    manifestId,
                                    request: SupplierManifestInjectOrderRequest(orderId: orderId, volumeVu: nil),
                                    idempotencyKey: idem("inject-order", extra: orderId)
                                )
                                injectOrderId = ""
                            }
                        }
                    }
                    .disabled(busy || injectOrderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                Section {
                    Button(busy ? "Sealing…" : "Seal manifest") {
                        Task { await runAction { try await SupplierOperationsService.sealManifest(manifestId, idempotencyKey: idem("seal")) } }
                    }
                    .disabled(busy)
                }
            }
            if !detail.orders.isEmpty {
                Section("Orders") {
                    ForEach(detail.orders) { order in
                        VStack(alignment: .leading) {
                            Text(order.orderId).font(.caption.monospaced())
                            Text(order.status.isEmpty ? order.state : order.status).font(.caption)
                        }
                    }
                }
            }
        }
    }

    private func idem(_ prefix: String, extra: String = "") -> String {
        "\(prefix):\(manifestId):\(extra.isEmpty ? UUID().uuidString : extra)"
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            detail = try await SupplierOperationsService.manifestDetail(manifestId)
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func runAction(_ action: () async throws -> Void) async {
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            try await action()
            await load()
        } catch {
            actionError = error.localizedDescription
        }
    }
}
