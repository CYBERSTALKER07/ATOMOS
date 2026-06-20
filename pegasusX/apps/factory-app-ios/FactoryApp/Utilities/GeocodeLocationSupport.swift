import Foundation

enum GeocodeLocationSupport {
    static func hasValidCoordinates(lat: Double, lng: Double) -> Bool {
        guard lat.isFinite, lng.isFinite else { return false }
        if lat == 0, lng == 0 { return false }
        return (-90 ... 90).contains(lat) && (-180 ... 180).contains(lng)
    }

    static func resolveLocationValue(_ value: AddressLocationValue) async -> AddressLocationValue? {
        let address = value.address.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !address.isEmpty else { return nil }
        if hasValidCoordinates(lat: value.lat, lng: value.lng) { return value }

        let predictions = (try? await GeocodeService.autocomplete(address)) ?? []
        if let top = predictions.first, !top.placeId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            if let byPlace = try? await GeocodeService.resolvePlace(top.placeId),
               hasValidCoordinates(lat: byPlace.lat, lng: byPlace.lng) {
                return AddressLocationValue(
                    address: byPlace.address.isEmpty ? address : byPlace.address,
                    lat: byPlace.lat,
                    lng: byPlace.lng,
                    placeId: byPlace.placeId
                )
            }
        }

        if let byAddress = try? await GeocodeService.forward(address: address),
           hasValidCoordinates(lat: byAddress.lat, lng: byAddress.lng) {
            return AddressLocationValue(
                address: byAddress.address.isEmpty ? address : byAddress.address,
                lat: byAddress.lat,
                lng: byAddress.lng,
                placeId: byAddress.placeId
            )
        }
        return nil
    }
}
