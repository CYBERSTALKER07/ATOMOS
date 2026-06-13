import SwiftUI

struct PricingView: View {
    @State private var rule: SupplierPricingRule?
    @State private var baseMarkupBps = ""
    @State private var retailerDiscountBps = ""
    @State private var minMarginBps = ""
    @State private var currency = ""
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading pricing…")
            } else if let error, rule == nil {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rule != nil {
                Form {
                    Section("Authority") {
                        TextField("Base markup (bps)", text: $baseMarkupBps)
                            .keyboardType(.numberPad)
                        TextField("Retailer discount (bps)", text: $retailerDiscountBps)
                            .keyboardType(.numberPad)
                        TextField("Min margin (bps)", text: $minMarginBps)
                            .keyboardType(.numberPad)
                        TextField("Currency", text: $currency)
                            .textInputAutocapitalization(.characters)
                    }
                    if let rule {
                        Section("Status") {
                            metricRow("Version", "\(rule.ruleVersion)")
                            metricRow("Updated", rule.updatedAt.isEmpty ? "—" : rule.updatedAt)
                        }
                    }
                    if let error {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }
                    Section {
                        Button(saving ? "Saving…" : "Save pricing rule") {
                            Task { await save() }
                        }
                        .disabled(saving)
                    }
                }
            } else {
                SupplierEmptyView(title: "No pricing rule", message: "Supplier pricing authority has not been configured.")
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Pricing")
        .task { await load() }
    }

    private func metricRow(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label).font(.caption).foregroundStyle(.secondary)
            Text(value).font(.headline)
        }
        .padding(.vertical, 4)
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let loaded = try await SupplierOperationsService.pricingRules()
            rule = loaded
            baseMarkupBps = String(loaded.baseMarkupBps)
            retailerDiscountBps = String(loaded.retailerDiscountBps)
            minMarginBps = String(loaded.minMarginBps)
            currency = loaded.currency
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func save() async {
        guard let base = Int64(baseMarkupBps.filter(\.isNumber)),
              let discount = Int64(retailerDiscountBps.filter(\.isNumber)),
              let margin = Int64(minMarginBps.filter(\.isNumber)),
              base >= 0, discount >= 0, margin >= 0 else {
            error = "Enter non-negative integer basis points"
            return
        }
        saving = true
        error = nil
        defer { saving = false }
        do {
            let trimmedCurrency = currency.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
            let updated = try await SupplierService.updatePricingRules(
                SupplierPricingRuleUpdateRequest(
                    baseMarkupBps: Int(base),
                    retailerDiscountBps: Int(discount),
                    minMarginBps: Int(margin),
                    currency: trimmedCurrency.count == 3 ? trimmedCurrency : nil
                )
            )
            rule = updated
            baseMarkupBps = String(updated.baseMarkupBps)
            retailerDiscountBps = String(updated.retailerDiscountBps)
            minMarginBps = String(updated.minMarginBps)
            currency = updated.currency
        } catch {
            self.error = error.localizedDescription
        }
    }
}
