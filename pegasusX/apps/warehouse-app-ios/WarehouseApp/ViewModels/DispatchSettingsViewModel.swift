import Foundation
import Observation

@Observable
@MainActor
final class DispatchSettingsViewModel {
    typealias FetchSettings = @MainActor () async throws -> DispatchSettingsResponse
    typealias PatchSettings = @MainActor (Bool) async throws -> Void

    var autoDispatchEnabled = false
    var hasLoaded = false
    var loading = true
    var saving = false
    var error: String?
    var saveMessage: String?

    private var lastSyncedValue: Bool?
    private let fetchSettings: FetchSettings
    private let patchSettings: PatchSettings

    init(
        fetchSettings: FetchSettings? = nil,
        patchSettings: PatchSettings? = nil
    ) {
        self.fetchSettings = fetchSettings ?? {
            try await WarehouseService.dispatchSettings()
        }
        self.patchSettings = patchSettings ?? { enabled in
            try await WarehouseService.patchDispatchSettings(enabled: enabled)
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let settings = try await fetchSettings()
            lastSyncedValue = settings.autoDispatchEnabled
            autoDispatchEnabled = settings.autoDispatchEnabled
            hasLoaded = true
        } catch {
            self.error = error.localizedDescription
        }
    }

    func handleToggleChange(_ newValue: Bool) async {
        guard hasLoaded, newValue != lastSyncedValue else { return }
        await save(enabled: newValue)
    }

    func save(enabled: Bool) async {
        saving = true
        saveMessage = nil
        defer { saving = false }
        do {
            try await patchSettings(enabled)
            lastSyncedValue = enabled
            autoDispatchEnabled = enabled
            hasLoaded = true
            saveMessage = enabled ? "Auto dispatch enabled" : "Auto dispatch disabled"
            PegasusHaptics.success()
        } catch {
            if let previous = lastSyncedValue {
                autoDispatchEnabled = previous
            }
            saveMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
