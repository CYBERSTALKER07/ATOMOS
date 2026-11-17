import Foundation
import Observation

@Observable
@MainActor
final class RescuesViewModel {
    typealias FetchDispatchPreview = @MainActor () async throws -> DispatchPreview
    typealias PreviewRescue = @MainActor (String) async throws -> RescuePreviewResponse
    typealias ProposeRescue = @MainActor (String, String, String) async throws -> [String: String]

    var brokenDrivers: [AvailableDriver] = []
    var selectedBroken: AvailableDriver?
    var rescueOptions: [RescueOption] = []
    var loading = true
    var previewLoading = false
    var proposeLoading = false
    var error: String?
    var statusMessage: String?

    private let fetchDispatchPreview: FetchDispatchPreview
    private let previewRescueFn: PreviewRescue
    private let proposeRescueFn: ProposeRescue

    init(
        fetchDispatchPreview: FetchDispatchPreview? = nil,
        previewRescue: PreviewRescue? = nil,
        proposeRescue: ProposeRescue? = nil
    ) {
        self.fetchDispatchPreview = fetchDispatchPreview ?? {
            try await WarehouseService.dispatchPreview()
        }
        self.previewRescueFn = previewRescue ?? { brokenDriverId in
            try await WarehouseService.previewRescue(brokenDriverId: brokenDriverId)
        }
        self.proposeRescueFn = proposeRescue ?? { rescueId, brokenDriverId, rescueDriverId in
            try await WarehouseService.proposeRescue(
                rescueId: rescueId,
                brokenDriverId: brokenDriverId,
                rescueDriverId: rescueDriverId
            )
        }
    }

    func loadDrivers() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let preview = try await fetchDispatchPreview()
            brokenDrivers = (preview.availableDrivers + preview.unavailableDrivers)
                .filter { $0.truckStatus.uppercased() == "NEEDS_RESCUE" }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func findRescue(_ driver: AvailableDriver) async {
        selectedBroken = driver
        rescueOptions = []
        previewLoading = true
        statusMessage = nil
        defer { previewLoading = false }
        do {
            let resp = try await previewRescueFn(driver.driverId)
            rescueOptions = resp.rescueOptions
            PegasusHaptics.light()
        } catch {
            statusMessage = error.localizedDescription
            selectedBroken = nil
            PegasusHaptics.error()
        }
    }

    func propose(_ option: RescueOption) async {
        guard let broken = selectedBroken else { return }
        proposeLoading = true
        defer { proposeLoading = false }
        do {
            let rescueId = UUID().uuidString
            _ = try await proposeRescueFn(rescueId, broken.driverId, option.driverId)
            statusMessage = "Rescue proposed"
            selectedBroken = nil
            rescueOptions = []
            PegasusHaptics.success()
            await loadDrivers()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
