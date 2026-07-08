//
//  ManifestService.swift
//  driverappios
//

import Foundation

// MARK: - Protocol

protocol ManifestServiceProtocol {
    /// GET /v1/driver/manifest, Authorization: Bearer {token}
    func downloadManifest(bearerToken: String) async throws -> RouteManifest
}

