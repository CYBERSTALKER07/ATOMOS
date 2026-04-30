//
//  SyncServiceLive.swift
//  driverappios
//
//  Real batch sync service for uploading offline deliveries to backend.
//

import Foundation

final class SyncServiceLive: SyncServiceProtocol {

    static let shared = SyncServiceLive()

    // MARK: - Upload Batch

    func uploadBatch(
        driverId: String,
        deliveries: [SyncDeliveryDTO],
        bearerToken: String
    ) async throws -> SyncResult {
        let body = BatchUploadRequest(
            driverId: driverId,
            deliveries: deliveries.map {
                BatchDelivery(
                    orderId: $0.orderId,
                    signature: $0.signature,
                    timestamp: $0.timestamp,
                    status: $0.status
                )
            }
        )

        let url = URL(string: "\(APIClient.shared.apiBaseURL)/v1/sync/batch")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            throw APIError.httpError((response as? HTTPURLResponse)?.statusCode ?? 500)
        }

        return try JSONDecoder().decode(SyncResult.self, from: data)
    }
}

// MARK: - Request DTOs

private struct BatchUploadRequest: Encodable {
    let driverId: String
    let deliveries: [BatchDelivery]

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case deliveries
    }
}

private struct BatchDelivery: Encodable {
    let orderId: String
    let signature: String
    let timestamp: Double
    let status: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case signature
        case timestamp
        case status
    }
}
