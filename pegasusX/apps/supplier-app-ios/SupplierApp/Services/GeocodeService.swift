import CoreLocation
import Foundation

struct GeocodePrediction: Decodable, Equatable {
  let placeId: String
  let description: String

  init(placeId: String, description: String) {
    self.placeId = placeId
    self.description = description
  }

  enum CodingKeys: String, CodingKey {
    case placeId = "place_id"
    case description
  }
}

struct ResolvedLocation: Decodable, Equatable {
  let address: String
  let lat: Double
  let lng: Double
  let placeId: String?

  init(address: String, lat: Double, lng: Double, placeId: String? = nil) {
    self.address = address
    self.lat = lat
    self.lng = lng
    self.placeId = placeId
  }

  enum CodingKeys: String, CodingKey {
    case address, lat, lng
    case placeId = "place_id"
  }
}

enum GeocodeService {
  static func autocomplete(_ input: String) async throws -> [GeocodePrediction] {
    let trimmed = input.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.count >= 3 else { return [] }
    let resp: AutocompleteResponse = try await APIClient.shared.get(
      "v1/platform/geocode/autocomplete",
      query: ["input": trimmed]
    )
    return resp.predictions
  }

  static func resolvePlace(_ placeId: String) async throws -> ResolvedLocation {
    let resolved: ResolvedLocation = try await APIClient.shared.get(
      "v1/platform/geocode/place",
      query: ["place_id": placeId]
    )
    guard !resolved.address.isEmpty, resolved.lat != 0 || resolved.lng != 0 else {
      throw APIError.server("geocode_not_found")
    }
    return resolved
  }

  static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
    if let result: ResolvedLocation = try? await APIClient.shared.get(
      "v1/platform/geocode/reverse",
      query: [
        "lat": String(lat),
        "lng": String(lng),
      ]
    ), !result.address.isEmpty {
      return result
    }

    let location = CLLocation(latitude: lat, longitude: lng)
    if let placemarks = try? await CLGeocoder().reverseGeocodeLocation(location),
       let p = placemarks.first {
      let parts = [p.name, p.thoroughfare, p.subLocality, p.locality, p.administrativeArea]
        .compactMap { $0 }
        .filter { !$0.isEmpty }
      if !parts.isEmpty {
        return ResolvedLocation(address: parts.joined(separator: ", "), lat: lat, lng: lng, placeId: nil)
      }
    }

    return ResolvedLocation(
      address: String(format: "Location (%.4f, %.4f)", lat, lng),
      lat: lat,
      lng: lng,
      placeId: nil
    )
  }

  static func forward(address: String) async throws -> ResolvedLocation {
    struct ForwardRequest: Encodable { let address: String }
    let trimmed = address.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { throw APIError.server("address_required") }
    if let result: ResolvedLocation = try? await APIClient.shared.post(
      "v1/platform/geocode/forward",
      body: ForwardRequest(address: trimmed)
    ), !result.address.isEmpty, result.lat != 0 || result.lng != 0 {
      return result
    }

    if let placemarks = try? await CLGeocoder().geocodeAddressString(trimmed),
       let p = placemarks.first,
       let loc = p.location {
      let parts = [p.name, p.thoroughfare, p.subLocality, p.locality, p.administrativeArea]
        .compactMap { $0 }
        .filter { !$0.isEmpty }
      let resolvedAddress = parts.isEmpty ? trimmed : parts.joined(separator: ", ")
      return ResolvedLocation(address: resolvedAddress, lat: loc.coordinate.latitude, lng: loc.coordinate.longitude, placeId: nil)
    }

    throw APIError.server("geocode_not_found")
  }

  private struct AutocompleteResponse: Decodable {
    let predictions: [GeocodePrediction]
  }
}
