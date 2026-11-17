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

    @State private var acceptBackorder: Bool

    init(
        preorderMinLeadDays: Binding<String>,
        preorderMaxLeadDays: Binding<String>,
        policy: Binding<String>,
        showStockCounts: Binding<Bool>,
        clearOrderLineMin: Binding<Bool>,
        orderLineMin: Binding<String>,
        clearOrderLineMax: Binding<Bool>,
        orderLineMax: Binding<String>,
        expressEnabled: Binding<Bool>,
        expressStockFloor: Binding<String>,
        clearFeeRules: Binding<Bool>,
        feeBaseMinor: Binding<String>,
        feeCurrency: Binding<String>,
        feeTiers: Binding<[FeeTierDraft]>,
        enforceOrderAcceptance: Binding<Bool>,
        scheduleIs24h: Binding<Bool>,
        scheduleTimezone: Binding<String>,
        weekdayOpen: Binding<String>,
        weekdayClose: Binding<String>,
        scheduleJSON: Binding<String>,
        scheduleError: String?
    ) {
        self._preorderMinLeadDays = preorderMinLeadDays
        self._preorderMaxLeadDays = preorderMaxLeadDays
        self._policy = policy
        self._showStockCounts = showStockCounts
        self._clearOrderLineMin = clearOrderLineMin
        self._orderLineMin = orderLineMin
        self._clearOrderLineMax = clearOrderLineMax
        self._orderLineMax = orderLineMax
        self._expressEnabled = expressEnabled
        self._expressStockFloor = expressStockFloor
        self._clearFeeRules = clearFeeRules
        self._feeBaseMinor = feeBaseMinor
        self._feeCurrency = feeCurrency
        self._feeTiers = feeTiers
        self._enforceOrderAcceptance = enforceOrderAcceptance
        self._scheduleIs24h = scheduleIs24h
        self._scheduleTimezone = scheduleTimezone
        self._weekdayOpen = weekdayOpen
        self._weekdayClose = weekdayClose
        self._scheduleJSON = scheduleJSON
        self.scheduleError = scheduleError
        self._acceptBackorder = State(initialValue: policy.wrappedValue == "ACCEPT_BACKORDER")
    }

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
            Toggle("Accept when out of stock", isOn: $acceptBackorder)
                .frame(minHeight: 44)
                .onChange(of: acceptBackorder) { _, newValue in
                    let target = newValue ? "ACCEPT_BACKORDER" : "REJECT"
                    if policy != target {
                        PegasusHaptics.selection()
                        policy = target
                    }
                }
                .onChange(of: policy) { _, newValue in
                    let shouldAccept = (newValue == "ACCEPT_BACKORDER")
                    if acceptBackorder != shouldAccept {
                        acceptBackorder = shouldAccept
                    }
                }
            Picker("Policy", selection: $policy) {
                Text("mobile_warehouse.ui.reject").tag("REJECT")
                Text("supplier_portal.inventory.inventory_table.text.accept_backorder").tag("ACCEPT_BACKORDER")
            }
            .pickerStyle(.inline)
        }

        Section("Retailer catalog display") {
            Toggle("Show stock counts to retailers", isOn: $showStockCounts)
                .frame(minHeight: 44)
        }

        Section("Order line quantity limits") {
            Toggle("No minimum quantity", isOn: $clearOrderLineMin)
                .frame(minHeight: 44)
            if !clearOrderLineMin {
                TextField("mobile_warehouse.ui.minimum_quantity", text: $orderLineMin)
                    .keyboardType(.numberPad)
            }
            Toggle("No maximum quantity", isOn: $clearOrderLineMax)
                .frame(minHeight: 44)
            if !clearOrderLineMax {
                TextField("mobile_warehouse.ui.maximum_quantity", text: $orderLineMax)
                    .keyboardType(.numberPad)
            }
        }

        Section("Express delivery") {
            Toggle("Express enabled", isOn: $expressEnabled)
                .frame(minHeight: 44)
            TextField("mobile_warehouse.ui.express_stock_floor", text: $expressStockFloor)
                .keyboardType(.numberPad)
        }

        Section("Delivery fee rules") {
            Toggle("No delivery fee rules", isOn: $clearFeeRules)
                .frame(minHeight: 44)
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
                    PegasusHaptics.light()
                    feeTiers.append(FeeTierDraft())
                }
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                if feeTiers.count > 1 {
                    Button("mobile_warehouse.ui.remove_last_tier", role: .destructive) {
                        PegasusHaptics.warning()
                        feeTiers.removeLast()
                    }
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                }
            }
        }

        Section("Order acceptance hours") {
            Text("warehouse_portal.residual.text.when_enforcement_is_on_retailers_cannot_preview_or_create_orders")
                .font(.caption)
                .foregroundStyle(.secondary)
            Toggle("Enforce order acceptance hours", isOn: $enforceOrderAcceptance)
                .frame(minHeight: 44)
            Toggle("Open 24 hours", isOn: $scheduleIs24h)
                .frame(minHeight: 44)
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
