import SwiftUI



struct OpsSettingsView: View {
    @State private var policy = "REJECT"
    @State private var showStockCounts = false
    @State private var preorderMinLeadDays = "3"
    @State private var preorderMaxLeadDays = "90"
    @State private var orderLineMin = ""
    @State private var orderLineMax = ""
    @State private var clearOrderLineMin = true
    @State private var clearOrderLineMax = true
    @State private var expressEnabled = false
    @State private var expressStockFloor = "0"
    @State private var feeBaseMinor = "0"
    @State private var feeCurrency = ""
    @State private var feeTiers: [FeeTierDraft] = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
    @State private var clearFeeRules = true
    @State private var scheduleJSON = "{\n  \"is_24h\": true\n}"
    @State private var enforceOrderAcceptance = false
    @State private var scheduleIs24h = true
    @State private var scheduleTimezone = "UTC"
    @State private var weekdayOpen = "09:00"
    @State private var weekdayClose = "17:00"
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var scheduleError: String?
    @State private var saveMessage: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else {
                Form {
                    OpsSettingsForm(
                        preorderMinLeadDays: $preorderMinLeadDays,
                        preorderMaxLeadDays: $preorderMaxLeadDays,
                        policy: $policy,
                        showStockCounts: $showStockCounts,
                        clearOrderLineMin: $clearOrderLineMin,
                        orderLineMin: $orderLineMin,
                        clearOrderLineMax: $clearOrderLineMax,
                        orderLineMax: $orderLineMax,
                        expressEnabled: $expressEnabled,
                        expressStockFloor: $expressStockFloor,
                        clearFeeRules: $clearFeeRules,
                        feeBaseMinor: $feeBaseMinor,
                        feeCurrency: $feeCurrency,
                        feeTiers: $feeTiers,
                        enforceOrderAcceptance: $enforceOrderAcceptance,
                        scheduleIs24h: $scheduleIs24h,
                        scheduleTimezone: $scheduleTimezone,
                        weekdayOpen: $weekdayOpen,
                        weekdayClose: $weekdayClose,
                        scheduleJSON: $scheduleJSON,
                        scheduleError: scheduleError
                    )

                    if let saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                        }
                    }

                    Section {
                        Button(saving ? "Saving…" : "Save settings") { save() }
                            .disabled(saving)
                    }
                }
            }
        }
        .navigationTitle("mobile_warehouse.ui.ops_settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }



    private func applyScheduleFields(from schedule: [String: AnyCodable]) {
        if let v = schedule["enforce_order_acceptance"]?.value as? Bool { enforceOrderAcceptance = v }
        if let v = schedule["is_24h"]?.value as? Bool { scheduleIs24h = v }
        if let v = schedule["timezone"]?.value as? String { scheduleTimezone = v }
        if let weekdays = schedule["schedules"]?.value as? [String: Any],
           let mon = weekdays["monday"] as? [String: Any] {
            if let open = mon["open"] as? String { weekdayOpen = open }
            if let close = mon["close"] as? String { weekdayClose = close }
        }
    }

    private func buildScheduleForSave() -> [String: Any]? {
        guard let data = scheduleJSON.data(using: .utf8),
              var object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        let weekdayWindow: [String: String] = ["open": weekdayOpen, "close": weekdayClose]
        object["enforce_order_acceptance"] = enforceOrderAcceptance
        object["is_24h"] = scheduleIs24h
        object["timezone"] = scheduleTimezone
        object["schedules"] = [
            "monday": weekdayWindow,
            "tuesday": weekdayWindow,
            "wednesday": weekdayWindow,
            "thursday": weekdayWindow,
            "friday": weekdayWindow,
        ]
        return object
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let settings = try await WarehouseService.opsSettings()
                policy = settings.defaultOutOfStockPolicy == "ACCEPT_BACKORDER" ? "ACCEPT_BACKORDER" : "REJECT"
                showStockCounts = settings.showStockCountsToRetailers
                preorderMinLeadDays = String(settings.preorderMinLeadDays)
                preorderMaxLeadDays = String(settings.preorderMaxLeadDays)
                if let minQty = settings.orderLineMinQuantity {
                    orderLineMin = String(minQty)
                    clearOrderLineMin = false
                } else {
                    orderLineMin = ""
                    clearOrderLineMin = true
                }
                if let maxQty = settings.orderLineMaxQuantity {
                    orderLineMax = String(maxQty)
                    clearOrderLineMax = false
                } else {
                    orderLineMax = ""
                    clearOrderLineMax = true
                }
                expressEnabled = settings.expressEnabled
                expressStockFloor = String(settings.expressStockFloor)
                if let rules = settings.deliveryFeeRules {
                    feeBaseMinor = String(rules.baseFeeMinor)
                    feeCurrency = rules.currency
                    feeTiers = rules.tiers.map {
                        FeeTierDraft(
                            maxKm: $0.maxKm.map { String($0) } ?? "",
                            feeMinor: String($0.feeMinor)
                        )
                    }
                    if feeTiers.isEmpty {
                        feeTiers = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
                    }
                    clearFeeRules = false
                } else {
                    feeBaseMinor = "0"
                    feeCurrency = ""
                    feeTiers = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
                    clearFeeRules = true
                }
                if let schedule = settings.operatingSchedule {
                    scheduleJSON = schedule.prettyJSONString()
                    applyScheduleFields(from: schedule)
                }
                if let packCurrency = try? await WarehouseService.paymentConfig().currencyCode, !packCurrency.isEmpty {
                    feeCurrency = packCurrency
                }
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func save() {
        saving = true
        saveMessage = nil
        scheduleError = nil
        guard let object = buildScheduleForSave() else {
            scheduleError = "Operating schedule must be valid JSON"
            saving = false
            return
        }
        guard let minLead = Int64(preorderMinLeadDays),
              let maxLead = Int64(preorderMaxLeadDays) else {
            saveMessage = "Pre-order lead days must be valid numbers"
            saving = false
            return
        }
        let schedule = object.mapValues { AnyCodable($0) }
        let deliveryRules: DeliveryFeeRules? = clearFeeRules ? nil : DeliveryFeeRules(
            currency: feeCurrency.trimmingCharacters(in: .whitespacesAndNewlines),
            baseFeeMinor: Int64(feeBaseMinor) ?? 0,
            tiers: feeTiers.map {
                let trimmed = $0.maxKm.trimmingCharacters(in: .whitespacesAndNewlines)
                return DeliveryFeeTier(
                    maxKm: trimmed.isEmpty ? nil : Double(trimmed),
                    feeMinor: Int64($0.feeMinor) ?? 0
                )
            }
        )
        let patch = WarehouseOpsSettingsPatchRequest(
            defaultOutOfStockPolicy: policy,
            showStockCountsToRetailers: showStockCounts,
            operatingSchedule: schedule,
            preorderMinLeadDays: minLead,
            preorderMaxLeadDays: maxLead,
            orderLineMinQuantity: clearOrderLineMin ? nil : Int64(orderLineMin),
            orderLineMaxQuantity: clearOrderLineMax ? nil : Int64(orderLineMax),
            clearOrderLineMinQuantity: clearOrderLineMin ? true : nil,
            clearOrderLineMaxQuantity: clearOrderLineMax ? true : nil,
            expressEnabled: expressEnabled,
            expressStockFloor: Int64(expressStockFloor) ?? 0,
            deliveryFeeRules: deliveryRules,
            clearDeliveryFeeRules: clearFeeRules ? true : nil
        )
        Task {
            do {
                try await WarehouseService.patchOpsSettings(patch)
                saveMessage = "Warehouse settings saved"
                load()
            } catch {
                saveMessage = error.localizedDescription
            }
            saving = false
        }
    }
}
