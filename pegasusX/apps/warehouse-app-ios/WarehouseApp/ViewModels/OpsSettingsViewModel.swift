import Foundation
import Observation

@Observable
@MainActor
final class OpsSettingsViewModel {
    typealias FetchOpsSettings = @MainActor () async throws -> WarehouseOpsSettingsResponse
    typealias FetchPaymentConfig = @MainActor () async throws -> PaymentConfigResponse
    typealias PatchOpsSettings = @MainActor (WarehouseOpsSettingsPatchRequest) async throws -> Void

    var policy = "REJECT"
    var showStockCounts = false
    var preorderMinLeadDays = "3"
    var preorderMaxLeadDays = "90"
    var orderLineMin = ""
    var orderLineMax = ""
    var clearOrderLineMin = true
    var clearOrderLineMax = true
    var expressEnabled = false
    var expressStockFloor = "0"
    var feeBaseMinor = "0"
    var feeCurrency = ""
    var feeTiers: [FeeTierDraft] = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
    var clearFeeRules = true
    var scheduleJSON = "{\n  \"is_24h\": true\n}"
    var enforceOrderAcceptance = false
    var scheduleIs24h = true
    var scheduleTimezone = "UTC"
    var weekdayOpen = "09:00"
    var weekdayClose = "17:00"
    var loading = true
    var saving = false
    var error: String?
    var scheduleError: String?
    var saveMessage: String?

    private let fetchOpsSettings: FetchOpsSettings
    private let fetchPaymentConfig: FetchPaymentConfig
    private let patchOpsSettings: PatchOpsSettings

    init(
        fetchOpsSettings: FetchOpsSettings? = nil,
        fetchPaymentConfig: FetchPaymentConfig? = nil,
        patchOpsSettings: PatchOpsSettings? = nil
    ) {
        self.fetchOpsSettings = fetchOpsSettings ?? {
            try await WarehouseService.opsSettings()
        }
        self.fetchPaymentConfig = fetchPaymentConfig ?? {
            try await WarehouseService.paymentConfig()
        }
        self.patchOpsSettings = patchOpsSettings ?? { patch in
            try await WarehouseService.patchOpsSettings(patch)
        }
    }

    func applyScheduleFields(from schedule: [String: AnyCodable]) {
        if let v = schedule["enforce_order_acceptance"]?.value as? Bool { enforceOrderAcceptance = v }
        if let v = schedule["is_24h"]?.value as? Bool { scheduleIs24h = v }
        if let v = schedule["timezone"]?.value as? String { scheduleTimezone = v }
        if let weekdays = schedule["schedules"]?.value as? [String: Any],
           let mon = weekdays["monday"] as? [String: Any] {
            if let open = mon["open"] as? String { weekdayOpen = open }
            if let close = mon["close"] as? String { weekdayClose = close }
        }
    }

    func buildScheduleForSave() -> [String: Any]? {
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

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let settings = try await fetchOpsSettings()
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
            if let packCurrency = try? await fetchPaymentConfig().currencyCode, !packCurrency.isEmpty {
                feeCurrency = packCurrency
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func save() async {
        saving = true
        saveMessage = nil
        scheduleError = nil
        defer { saving = false }

        guard let object = buildScheduleForSave() else {
            scheduleError = "Operating schedule must be valid JSON"
            PegasusHaptics.warning()
            return
        }
        guard let minLead = Int64(preorderMinLeadDays),
              let maxLead = Int64(preorderMaxLeadDays) else {
            saveMessage = "Pre-order lead days must be valid numbers"
            PegasusHaptics.warning()
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
        do {
            try await patchOpsSettings(patch)
            saveMessage = "Warehouse settings saved"
            PegasusHaptics.success()
            await load()
        } catch {
            saveMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
