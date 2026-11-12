import SwiftUI

struct CreditPartnersView: View {
    @State private var vm = CreditPartnersViewModel()

    var body: some View {
        List {
            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                }
            }
            Section("Trade credit partners") {
                if vm.relationships.isEmpty {
                    Text("No credit relationships.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(vm.relationships, id: \.supplierId) { r in
                        let limitFormatted = RetailerMoney.formatTiyin(r.creditLimitMinor)
                        let balanceFormatted = RetailerMoney.formatTiyin(r.currentBalanceMinor)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(r.supplierId)
                                .font(.headline)
                            Text("Limit \(limitFormatted) · Balance \(balanceFormatted)")
                                .font(.caption)
                            Text("Terms \(r.termsDays)d\(r.onHold ? " · ON HOLD" : "")")
                                .font(.caption2)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        .padding(.vertical, 2)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(
                            "Supplier \(r.supplierId), limit \(limitFormatted), balance \(balanceFormatted), terms \(r.termsDays) days\(r.onHold ? ", on hold" : "")"
                        )
                    }
                }
            }
            Section("Open AR invoices") {
                if vm.invoices.isEmpty {
                    Text("No open invoices.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(vm.invoices, id: \.invoiceId) { inv in
                        let balanceFormatted = RetailerMoney.formatTiyin(inv.balanceMinor)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(inv.invoiceId)
                                .font(.headline)
                            Text("\(inv.supplierId) · \(inv.status)")
                                .font(.caption)
                            Text("Balance \(balanceFormatted) · Due \(inv.dueAt)")
                                .font(.caption2)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        .padding(.vertical, 2)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(
                            "Invoice \(inv.invoiceId), supplier \(inv.supplierId), status \(inv.status), balance \(balanceFormatted), due \(inv.dueAt)"
                        )
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("Credit & AR")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
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
