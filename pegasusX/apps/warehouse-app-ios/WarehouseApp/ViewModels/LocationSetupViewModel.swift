import Foundation
import Observation

@Observable
@MainActor
final class LocationSetupViewModel {
    var warehouseName = ""
    var location = AddressLocationValue()
    var loading = false
    var submitting = false
    var error: String?
    var packCountry = ""

    private let fetchLocationHandler: @Sendable () async throws -> WarehouseLocationResponse
    private let resolveLocationHandler: @MainActor (AddressLocationValue) async -> AddressLocationValue?
    private let patchLocationHandler: @Sendable (String, String?, Double, Double) async throws -> WarehouseLocationResponse
    private let refreshTokenHandler: @Sendable (String) async throws -> AuthResponse
    private let setupHandler: @Sendable (String, String, String?, Double, Double) async throws -> WarehouseSetupResponse

    init(
        fetchLocationHandler: @escaping @Sendable () async throws -> WarehouseLocationResponse = {
            try await WarehouseService.warehouseLocation()
        },
        resolveLocationHandler: @escaping @MainActor (AddressLocationValue) async -> AddressLocationValue? = { value in
            await GeocodeLocationSupport.resolveLocationValue(value)
        },
        patchLocationHandler: @escaping @Sendable (String, String?, Double, Double) async throws -> WarehouseLocationResponse = { address, placeId, lat, lng in
            try await WarehouseService.patchWarehouseLocation(address: address, placeId: placeId, lat: lat, lng: lng)
        },
        refreshTokenHandler: @escaping @Sendable (String) async throws -> AuthResponse = { refresh in
            try await WarehouseOperationsService.refreshToken(refresh)
        },
        setupHandler: @escaping @Sendable (String, String, String?, Double, Double) async throws -> WarehouseSetupResponse = { name, address, placeId, lat, lng in
            try await WarehouseService.setup(name: name, address: address, placeId: placeId, lat: lat, lng: lng)
        }
    ) {
        self.fetchLocationHandler = fetchLocationHandler
        self.resolveLocationHandler = resolveLocationHandler
        self.patchLocationHandler = patchLocationHandler
        self.refreshTokenHandler = refreshTokenHandler
        self.setupHandler = setupHandler
    }

    func loadExistingIfNeeded(hasAssignedWarehouse: Bool) async {
        guard hasAssignedWarehouse else { return }
        loading = true
        defer { loading = false }
        do {
            let resp = try await fetchLocationHandler()
            warehouseName = resp.name
            location = AddressLocationValue(
                address: resp.address,
                lat: resp.lat,
                lng: resp.lng,
                placeId: resp.placeId
            )
            packCountry = resp.packCountryCode.isEmpty ? resp.countryCode : resp.packCountryCode
            error = nil
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submit(tokenStore: TokenStore) async {
        submitting = true
        error = nil
        defer { submitting = false }

        guard let resolved = await resolveLocationHandler(location) else {
            error = "Select an address from the suggestions or share your location."
            PegasusHaptics.error()
            return
        }
        location = resolved

        do {
            if tokenStore.hasAssignedWarehouse {
                _ = try await patchLocationHandler(
                    resolved.address,
                    resolved.placeId,
                    resolved.lat,
                    resolved.lng
                )
                if let refresh = tokenStore.refreshToken {
                    let auth = try await refreshTokenHandler(refresh)
                    tokenStore.updateTokens(
                        token: auth.token,
                        refresh: auth.refreshToken,
                        warehouseId: auth.warehouseId
                    )
                }
                PegasusHaptics.success()
            } else {
                let trimmed = warehouseName.trimmingCharacters(in: .whitespacesAndNewlines)
                guard trimmed.count >= 3 else {
                    error = "Warehouse name is required."
                    PegasusHaptics.error()
                    return
                }
                let resp = try await setupHandler(
                    trimmed,
                    resolved.address,
                    resolved.placeId,
                    resolved.lat,
                    resolved.lng
                )
                guard let token = resp.token, !token.isEmpty else {
                    error = "Setup failed."
                    PegasusHaptics.error()
                    return
                }
                tokenStore.updateTokens(
                    token: token,
                    refresh: resp.refreshToken ?? tokenStore.refreshToken ?? "",
                    warehouseId: resp.warehouseId
                )
                PegasusHaptics.success()
            }
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
