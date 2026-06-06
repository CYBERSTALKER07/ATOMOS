//
//  ManifestServiceLive.swift
//  driverappios
//
//  Real manifest service backed by APIClient.
//

import Foundation

final class ManifestServiceLive: ManifestServiceProtocol {

    static let shared = ManifestServiceLive()

    func downloadManifest(bearerToken: String) async throws -> RouteManifest {
        let today = ISO8601DateFormatter().string(from: Date()).prefix(10)
        return try await APIClient.shared.getManifest(date: String(today))
    }
}
