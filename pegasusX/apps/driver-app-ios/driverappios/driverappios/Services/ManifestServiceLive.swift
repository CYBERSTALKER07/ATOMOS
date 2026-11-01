//
//  ManifestServiceLive.swift
//  driverappios
//
//  Real manifest service backed by APIClient.
//

import Foundation

final class ManifestServiceLive: ManifestServiceProtocol {

    static let shared = ManifestServiceLive()
    private static let isoFormatter = ISO8601DateFormatter()

    func downloadManifest(bearerToken: String) async throws -> RouteManifest {
        let today = Self.isoFormatter.string(from: Date()).prefix(10)
        return try await APIClient.shared.getManifest(date: String(today))
    }
}
