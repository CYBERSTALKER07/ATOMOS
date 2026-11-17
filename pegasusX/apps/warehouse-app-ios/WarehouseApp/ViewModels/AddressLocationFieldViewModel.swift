import CoreLocation
import Foundation
import Observation

@Observable
@MainActor
final class AddressLocationFieldViewModel {
    var query = ""
    var suggestions: [GeocodePrediction] = []
    var error: String?
    var locating = false
    var resolving = false

    private var searchTask: Task<Void, Never>?

    private let autocompleteHandler: @Sendable (String) async throws -> [GeocodePrediction]
    private let resolvePlaceHandler: @Sendable (String) async throws -> ResolvedLocation
    private let reverseHandler: @Sendable (Double, Double) async throws -> ResolvedLocation
    private let resolveLocationValueHandler: @MainActor (AddressLocationValue) async -> AddressLocationValue?
    private let coordinateProvider: @MainActor () -> CLLocationCoordinate2D?

    init(
        autocompleteHandler: @escaping @Sendable (String) async throws -> [GeocodePrediction] = { text in
            try await GeocodeService.autocomplete(text)
        },
        resolvePlaceHandler: @escaping @Sendable (String) async throws -> ResolvedLocation = { placeId in
            try await GeocodeService.resolvePlace(placeId)
        },
        reverseHandler: @escaping @Sendable (Double, Double) async throws -> ResolvedLocation = { lat, lng in
            try await GeocodeService.reverse(lat: lat, lng: lng)
        },
        resolveLocationValueHandler: @escaping @MainActor (AddressLocationValue) async -> AddressLocationValue? = { value in
            await GeocodeLocationSupport.resolveLocationValue(value)
        },
        coordinateProvider: @escaping @MainActor () -> CLLocationCoordinate2D? = {
            let manager = CLLocationManager()
            manager.requestWhenInUseAuthorization()
            return manager.location?.coordinate
        }
    ) {
        self.autocompleteHandler = autocompleteHandler
        self.resolvePlaceHandler = resolvePlaceHandler
        self.reverseHandler = reverseHandler
        self.resolveLocationValueHandler = resolveLocationValueHandler
        self.coordinateProvider = coordinateProvider
    }

    func syncExternalAddress(_ address: String) {
        if query != address {
            query = address
        }
    }

    func scheduleSearch(_ text: String, debounceNanoseconds: UInt64 = 280_000_000) {
        searchTask?.cancel()
        searchTask = Task { @MainActor in
            if debounceNanoseconds > 0 {
                try? await Task.sleep(nanoseconds: debounceNanoseconds)
            }
            guard !Task.isCancelled else { return }
            await search(text)
        }
    }

    func cancelSearch() {
        searchTask?.cancel()
        searchTask = nil
    }

    func search(_ text: String) async {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 3 else {
            suggestions = []
            return
        }
        suggestions = (try? await autocompleteHandler(trimmed)) ?? []
    }

    func pick(placeId: String, fallback: String, currentValue: AddressLocationValue) async -> AddressLocationValue {
        let id = placeId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            return await resolveText(fallback, currentValue: currentValue)
        }
        do {
            let loc = try await resolvePlaceHandler(id)
            return apply(loc, fallback: fallback)
        } catch {
            return await resolveText(fallback, currentValue: currentValue)
        }
    }

    func resolveOnBlur(currentValue: AddressLocationValue) async -> AddressLocationValue {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return currentValue }
        if trimmed == currentValue.address.trimmingCharacters(in: .whitespacesAndNewlines),
           GeocodeLocationSupport.hasValidCoordinates(lat: currentValue.lat, lng: currentValue.lng) {
            return currentValue
        }
        return await resolveText(trimmed, currentValue: currentValue)
    }

    func resolveText(_ text: String, currentValue: AddressLocationValue) async -> AddressLocationValue {
        resolving = true
        defer { resolving = false }
        if let resolved = await resolveLocationValueHandler(
            AddressLocationValue(address: text, lat: currentValue.lat, lng: currentValue.lng, placeId: currentValue.placeId)
        ) {
            query = resolved.address
            suggestions = []
            error = nil
            return resolved
        }
        return currentValue
    }

    func useMyLocation(currentValue: AddressLocationValue) async -> AddressLocationValue {
        locating = true
        defer { locating = false }
        var coord = coordinateProvider()
        if coord == nil || (coord?.latitude == 0 && coord?.longitude == 0) {
            coord = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        }
        guard let target = coord else {
            error = "Current location unavailable."
            return currentValue
        }
        do {
            let loc = try await reverseHandler(target.latitude, target.longitude)
            return apply(loc, fallback: query.isEmpty ? "Amir Temur Ave, Tashkent" : query)
        } catch {
            return apply(
                ResolvedLocation(
                    address: "Amir Temur Ave, Tashkent, Uzbekistan",
                    lat: target.latitude,
                    lng: target.longitude,
                    placeId: nil
                ),
                fallback: "Amir Temur Ave, Tashkent, Uzbekistan"
            )
        }
    }

    private func apply(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
        let next = AddressLocationValue(
            address: loc.address.isEmpty ? fallback : loc.address,
            lat: loc.lat,
            lng: loc.lng,
            placeId: loc.placeId
        )
        query = next.address
        suggestions = []
        error = nil
        return next
    }
}
