import SwiftUI

struct CreditPartnersView: View {
    @State private var relationships: [CreditRelationshipWire] = []
    @State private var invoices: [ArInvoiceWire] = []
    @State private var banner: String?
    private let api = APIClient.shared

    var body: some View {
        List {
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Trade credit partners") {
                if relationships.isEmpty {
                    Text("No credit relationships.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(relationships, id: \.supplierId) { r in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(r.supplierId).font(.headline)
                            Text(String(format: "Limit %.2f · Balance %.2f", Double(r.creditLimitMinor) / 100.0, Double(r.currentBalanceMinor) / 100.0))
                                .font(.caption)
                            Text("Terms \(r.termsDays)d\(r.onHold ? " · ON HOLD" : "")")
                                .font(.caption2)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                    }
                }
            }
            Section("Open AR invoices") {
                if invoices.isEmpty {
                    Text("No open invoices.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(invoices, id: \.invoiceId) { inv in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(inv.invoiceId).font(.headline)
                            Text("\(inv.supplierId) · \(inv.status)")
                                .font(.caption)
                            Text(String(format: "Balance %.2f · Due %@", Double(inv.balanceMinor) / 100.0, inv.dueAt))
                                .font(.caption2)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                    }
                }
            }
        }
        .navigationTitle("Credit & AR")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        do {
            async let rels = api.getCreditRelationships()
            async let invs = api.getArInvoices(status: "OPEN")
            relationships = try await rels
            invoices = try await invs
            banner = nil
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct CreditRelationshipWire: Decodable {
    let supplierId: String
    let creditLimitMinor: Int64
    let currentBalanceMinor: Int64
    let termsDays: Int
    let onHold: Bool

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case creditLimitMinor = "credit_limit_minor"
        case currentBalanceMinor = "current_balance_minor"
        case termsDays = "terms_days"
        case onHold = "on_hold"
        case availableCreditMinor = "available_credit_minor"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? "—"
        creditLimitMinor = try c.decodeIfPresent(Int64.self, forKey: .creditLimitMinor) ?? 0
        currentBalanceMinor = try c.decodeIfPresent(Int64.self, forKey: .currentBalanceMinor) ?? 0
        termsDays = try c.decodeIfPresent(Int.self, forKey: .termsDays) ?? 0
        onHold = try c.decodeIfPresent(Bool.self, forKey: .onHold) ?? false
    }
}

struct CreditRelationshipsResponse: Decodable {
    let relationships: [CreditRelationshipWire]?
}

struct ArInvoiceWire: Decodable {
    let invoiceId: String
    let supplierId: String
    let balanceMinor: Int64
    let status: String
    let dueAt: String

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case supplierId = "supplier_id"
        case balanceMinor = "balance_minor"
        case status
        case dueAt = "due_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        invoiceId = try c.decodeIfPresent(String.self, forKey: .invoiceId) ?? "—"
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? "—"
        balanceMinor = try c.decodeIfPresent(Int64.self, forKey: .balanceMinor) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "—"
        dueAt = try c.decodeIfPresent(String.self, forKey: .dueAt) ?? "—"
    }
}

struct ArInvoicesResponse: Decodable {
    let invoices: [ArInvoiceWire]?
}
