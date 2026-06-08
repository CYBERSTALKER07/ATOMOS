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

// MARK: - Stub Implementation

final class ManifestServiceStub: ManifestServiceProtocol {

    static let shared = ManifestServiceStub()

    func downloadManifest(bearerToken: String) async throws -> RouteManifest {
        try await Task.sleep(nanoseconds: 500_000_000)
        return RouteManifest.mock
    }
}
