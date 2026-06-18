import Foundation

struct GeocodePrediction: Decodable {
  let placeId: String
  let description: String

  enum CodingKeys: String, CodingKey {
    case placeId = "place_id"
    case description
  }
}

struct ResolvedLocation: Decodable {
  let address: String
  let lat: Double
  let lng: Double
  let placeId: String?

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
    try await APIClient.shared.get(
      "v1/platform/geocode/place",
      query: ["place_id": placeId]
    )
  }

  static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
    try await APIClient.shared.get(
      "v1/platform/geocode/reverse",
      query: ["lat": String(lat), "lng": String(lng)]
    )
  }

  private struct AutocompleteResponse: Decodable {
    let predictions: [GeocodePrediction]
  }
}
