import Foundation
import Observation

struct EditableNotificationPreference: Identifiable, Equatable {
    let id: String
    let eventType: String
    let channel: String
    var enabled: Bool
}

@Observable
@MainActor
final class NotificationPreferencesViewModel {
    var items: [EditableNotificationPreference] = []
    var loading = true
    var saved = false
    var error: String?
    private var loadGeneration = 0

    func load() async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty
        loading = true
        defer {
            if generation == loadGeneration {
                loading = false
            }
        }
        do {
            let resp = try await SupplierOperationsService.notificationPreferences()
            guard generation == loadGeneration else { return }
            items = resp.preferences.enumerated().map { idx, row in
                EditableNotificationPreference(
                    id: "\(row.eventType)|\(row.channel)|\(idx)",
                    eventType: row.eventType,
                    channel: row.channel,
                    enabled: row.enabled
                )
            }
            error = nil
            SupplierStatusCenter.shared.notice = nil
        } catch {
            guard generation == loadGeneration else { return }
            if hadContent {
                SupplierStatusCenter.shared.notice = error.localizedDescription
            } else {
                self.error = error.localizedDescription
            }
        }
    }

    func save() async {
        let payload = items.map {
            NotificationPreferenceRow(eventType: $0.eventType, channel: $0.channel, enabled: $0.enabled)
        }
        do {
            try await SupplierOperationsService.patchNotificationPreferences(payload)
            saved = true
            error = nil
            PegasusHaptics.success()
            SupplierStatusCenter.shared.notice = nil
        } catch {
            saved = false
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: !items.isEmpty, error: &self.error)
        }
    }
}

@Observable
@MainActor
final class ReturnPolicySettingsViewModel {
    static let hourPresets: [Int64] = [8, 24, 48, 72]

    var loading = true
    var hasLoaded = false
    var saving = false
    var error: String?
    var saved = false
    var hours: Int64 = 48
    var hoursText = "48"
    var concealed = ""
    var requirePhoto = true
    var allowExpired = false
    var sourceHint = ""
    private var loadGeneration = 0

    func selectPreset(_ preset: Int64) {
        PegasusHaptics.selection()
        hours = preset
        hoursText = String(preset)
    }

    func sanitizeHoursText(_ newValue: String) {
        let digits = newValue.filter(\.isNumber)
        if digits != newValue { hoursText = digits }
        if let parsed = Int64(digits) { hours = parsed }
    }

    func sanitizeConcealed(_ newValue: String) {
        let digits = newValue.filter(\.isNumber)
        if digits != newValue { concealed = digits }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = hasLoaded
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let p = try await SupplierOperationsService.returnPolicy()
            guard generation == loadGeneration else { return }
            hours = p.defaultWindowHours > 0 ? p.defaultWindowHours : 48
            hoursText = String(hours)
            if let c = p.concealedDamageWindowHours, c > 0 {
                concealed = String(c)
            } else {
                concealed = ""
            }
            requirePhoto = p.requirePhoto
            allowExpired = p.allowExpiredClaims
            sourceHint = p.policySourceHint ?? ""
            hasLoaded = true
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func save() async {
        let parsed = Int64(hoursText) ?? hours
        guard parsed >= 1, parsed <= 168 else {
            error = L10n.format("mobile_supplier.chrome.claim_window_range")
            PegasusHaptics.error()
            return
        }
        saving = true
        error = nil
        saved = false
        defer { saving = false }
        var body = SupplierReturnPolicy(
            defaultWindowHours: parsed,
            concealedDamageWindowHours: nil,
            requirePhoto: requirePhoto,
            allowExpiredClaims: allowExpired,
            policySourceHint: nil
        )
        if let c = Int64(concealed.trimmingCharacters(in: .whitespaces)), c > 0 {
            body.concealedDamageWindowHours = c
        }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let key = SupplierIdempotencyKeys.returnPolicyPut(scopeId: scope, hours: parsed)
            let savedPolicy = try await SupplierOperationsService.putReturnPolicy(body, idempotencyKey: key)
            sourceHint = savedPolicy.policySourceHint ?? "SUPPLIER"
            hours = parsed
            hoursText = String(parsed)
            saved = true
            PegasusHaptics.success()
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
        }
    }
}
