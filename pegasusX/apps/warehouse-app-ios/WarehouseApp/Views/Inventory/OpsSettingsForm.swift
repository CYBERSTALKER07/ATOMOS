import SwiftUI

struct FeeTierDraft: Identifiable, Equatable {
    let id = UUID()
    var maxKm: String
    var feeMinor: String

    init(maxKm: String = "", feeMinor: String = "0") {
        self.maxKm = maxKm
        self.feeMinor = feeMinor
    }
}

struct OpsSettingsForm: View {
    @Binding var preorderMinLeadDays: String
    @Binding var preorderMaxLeadDays: String
    @Binding var policy: String
    @Binding var showStockCounts: Bool
    @Binding var clearOrderLineMin: Bool
    @Binding var orderLineMin: String
    @Binding var clearOrderLineMax: Bool
    @Binding var orderLineMax: String
    @Binding var expressEnabled: Bool
    @Binding var expressStockFloor: String
    @Binding var clearFeeRules: Bool
    @Binding var feeBaseMinor: String
    @Binding var feeCurrency: String
    @Binding var feeTiers: [FeeTierDraft]
    @Binding var enforceOrderAcceptance: Bool
    @Binding var scheduleIs24h: Bool
    @Binding var scheduleTimezone: String
    @Binding var weekdayOpen: String
    @Binding var weekdayClose: String
    @Binding var scheduleJSON: String
    let scheduleError: String?

    var body: some View {
        Section {
            Text("mobile_warehouse.ui.checkout_policy_pre_orders_delivery_fees_and_retailer_catalog_di")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }

        Section("Pre-order lead window") {
            Text("mobile_warehouse.ui.retailers_can_request_delivery_between_these_lead_days_from_toda")
                .font(.caption)
                .foregroundStyle(.secondary)
            HStack {
                TextField("mobile_warehouse.ui.min_days", text: $preorderMinLeadDays)
                    .keyboardType(.numberPad)
                TextField("mobile_warehouse.ui.max_days", text: $preorderMaxLeadDays)
                    .keyboardType(.numberPad)
            }
        }

        Section("Out-of-stock orders") {
            Toggle("Accept when out of stock", isOn: Binding(
                get: { policy == "ACCEPT_BACKORDER" },
                set: { policy = $0 ? "ACCEPT_BACKORDER" : "REJECT" }
            ))
            Picker("Policy", selection: $policy) {
                Text("mobile_warehouse.ui.reject").tag("REJECT")
                Text("supplier_portal.inventory.inventory_table.text.accept_backorder").tag("ACCEPT_BACKORDER")
            }
            .pickerStyle(.inline)
        }

        Section("Retailer catalog display") {
            Toggle("Show stock counts to retailers", isOn: $showStockCounts)
        }

        Section("Order line quantity limits") {
            Toggle("No minimum quantity", isOn: $clearOrderLineMin)
            if !clearOrderLineMin {
                TextField("mobile_warehouse.ui.minimum_quantity", text: $orderLineMin)
                    .keyboardType(.numberPad)
            }
            Toggle("No maximum quantity", isOn: $clearOrderLineMax)
            if !clearOrderLineMax {
                TextField("mobile_warehouse.ui.maximum_quantity", text: $orderLineMax)
                    .keyboardType(.numberPad)
            }
        }

        Section("Express delivery") {
            Toggle("Express enabled", isOn: $expressEnabled)
            TextField("mobile_warehouse.ui.express_stock_floor", text: $expressStockFloor)
                .keyboardType(.numberPad)
        }

        Section("Delivery fee rules") {
            Toggle("No delivery fee rules", isOn: $clearFeeRules)
            if !clearFeeRules {
                TextField("warehouse_portal.residual.text.base_fee_minor", text: $feeBaseMinor)
                    .keyboardType(.numberPad)
                LabeledContent("supplier_portal.chargebacks.text.currency", value: feeCurrency)
                ForEach($feeTiers) { $tier in
                    HStack {
                        TextField("mobile_warehouse.ui.max_km", text: $tier.maxKm)
                            .keyboardType(.decimalPad)
                        TextField("mobile_warehouse.ui.fee_minor", text: $tier.feeMinor)
                            .keyboardType(.numberPad)
                    }
                }
                Button("mobile_warehouse.ui.add_tier") {
                    feeTiers.append(FeeTierDraft())
                }
                if feeTiers.count > 1 {
                    Button("mobile_warehouse.ui.remove_last_tier", role: .destructive) {
                        feeTiers.removeLast()
                    }
                }
            }
        }

        Section("Order acceptance hours") {
            Text("warehouse_portal.residual.text.when_enforcement_is_on_retailers_cannot_preview_or_create_orders")
                .font(.caption)
                .foregroundStyle(.secondary)
            Toggle("Enforce order acceptance hours", isOn: $enforceOrderAcceptance)
            Toggle("Open 24 hours", isOn: $scheduleIs24h)
            TextField("supplier_portal.configuration.countries.field.timezone", text: $scheduleTimezone)
            HStack {
                TextField("warehouse_portal.residual.text.weekday_open", text: $weekdayOpen)
                TextField("warehouse_portal.residual.text.weekday_close", text: $weekdayClose)
            }
            Text("warehouse_portal.settings.ops_settings_form.text.advanced_json").font(.caption).foregroundStyle(.secondary)
            TextEditor(text: $scheduleJSON)
                .font(.system(.caption, design: .monospaced))
                .frame(minHeight: 140)
            if let scheduleError {
                Text(scheduleError).foregroundStyle(.red).font(.caption)
            }
        }
    }
}
