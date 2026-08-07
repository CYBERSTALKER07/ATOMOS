import SwiftUI

struct LocalSkusView: View {
    @State private var rows: [LocalSkuRowIOS] = []
    @State private var name = ""
    @State private var barcode = ""
    @State private var price = "5000"
    @State private var banner: String?
    @State private var busy = false
    private let api = APIClient.shared

    var body: some View {
        List {
            if let banner { Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) } }
            Section("Add local SKU") {
                TextField("retailer_desktop.pos.text.name", text: $name)
                TextField("retailer_desktop.stock.local_skus.text.barcode", text: $barcode)
                TextField("supplier_portal.catalog.components.catalog_table.text.price_minor", text: $price)
                Button(busy ? "…" : "Create") { Task { await create() } }
                    .disabled(busy || name.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            Section("Catalog") {
                if rows.isEmpty {
                    Text("mobile_retailer.ui.no_local_skus").foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(rows) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.name).font(.headline)
                            Text("\(row.id) · \(row.priceMinor) · \(row.active ? "active" : "inactive")")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textSecondary)
                            Button(row.active ? "Disable" : "Enable") {
                                Task { await toggle(row) }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("portal.nav.local_skus")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        do {
            let list = try await api.getLocalSkus()
            rows = list.items.map {
                LocalSkuRowIOS(
                    id: $0.localSkuId,
                    name: $0.name,
                    priceMinor: $0.defaultPriceMinor ?? 0,
                    active: $0.isActive ?? true
                )
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createLocalSku(
                name: name.trimmingCharacters(in: .whitespaces),
                barcode: barcode.trimmingCharacters(in: .whitespaces),
                priceMinor: Int64(price) ?? 0
            )
            name = ""
            barcode = ""
            banner = "Created"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func toggle(_ row: LocalSkuRowIOS) async {
        do {
            _ = try await api.patchLocalSku(id: row.id, isActive: !row.active)
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct LocalSkuRowIOS: Identifiable {
    let id: String
    let name: String
    let priceMinor: Int64
    let active: Bool
}
