import Foundation
import Observation

@Observable
@MainActor
final class LocationSettingsViewModel {
    typealias FetchLocation = @MainActor () async throws -> WarehouseLocationResponse
    typealias ResolveLocation = @MainActor (AddressLocationValue) async -> AddressLocationValue?
    typealias PatchLocation = @MainActor (String, String?, Double, Double) async throws -> WarehouseLocationResponse

    var warehouseName = ""
    var location = AddressLocationValue()
    var loading = true
    var saving = false
    var error: String?
    var saveMessage: String?
    var packCountry = ""

    private let fetchLocation: FetchLocation
    private let resolveLocation: ResolveLocation
    private let patchLocation: PatchLocation

    init(
        fetchLocation: FetchLocation? = nil,
        resolveLocation: ResolveLocation? = nil,
        patchLocation: PatchLocation? = nil
    ) {
        self.fetchLocation = fetchLocation ?? {
            try await WarehouseService.warehouseLocation()
        }
        self.resolveLocation = resolveLocation ?? { value in
            await GeocodeLocationSupport.resolveLocationValue(value)
        }
        self.patchLocation = patchLocation ?? { address, placeId, lat, lng in
            try await WarehouseService.patchWarehouseLocation(
                address: address,
                placeId: placeId,
                lat: lat,
                lng: lng
            )
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchLocation()
            warehouseName = resp.name
            location = AddressLocationValue(
                address: resp.address,
                lat: resp.lat,
                lng: resp.lng,
                placeId: resp.placeId
            )
            packCountry = resp.packCountryCode.isEmpty ? resp.countryCode : resp.packCountryCode
        } catch {
            self.error = error.localizedDescription
        }
    }

    func save() async {
        saving = true
        saveMessage = nil
        defer { saving = false }
        guard let resolved = await resolveLocation(location) else {
            saveMessage = "Select an address from the suggestions or share your location."
            PegasusHaptics.warning()
            return
        }
        location = resolved
        do {
            _ = try await patchLocation(
                resolved.address,
                resolved.placeId,
                resolved.lat,
                resolved.lng
            )
            saveMessage = "Location saved"
            PegasusHaptics.success()
            await load()
        } catch let saveError {
            saveMessage = saveError.localizedDescription
            PegasusHaptics.error()
        }
    }
}
