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
        var processed: [String] = []
        var skipped = 0

        for delivery in deliveries {
            do {
                let synced = try await uploadDeliveryMutation(delivery, bearerToken: bearerToken)
                if synced {
                    processed.append(delivery.orderId)
                } else {
                    skipped += 1
                }
            } catch let error as APIError {
                if case .httpError(409) = error {
                    processed.append(delivery.orderId)
                    skipped += 1
                } else {
                    throw error
                }
            }
        }

        return SyncResult(status: "SYNC_COMPLETE", processed: processed, skipped: skipped)
    }

    // MARK: - Per-delivery mutation (Idempotency-Key header)

    private func uploadDeliveryMutation(_ delivery: SyncDeliveryDTO, bearerToken: String) async throws -> Bool {
        let body: [String: Any] = [
            "order_id": delivery.orderId,
            "qr_token": delivery.signature,
            "latitude": 0.0,
            "longitude": 0.0,
        ]
        let url = URL(string: "\(APIClient.shared.apiBaseURL)/v1/order/deliver")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        request.setValue(delivery.idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.httpError(500)
        }
        if http.statusCode == 409 {
            throw APIError.httpError(409)
        }
        guard (200...299).contains(http.statusCode) else {
            throw APIError.httpError(http.statusCode)
        }
        _ = data
        return true
    }
}

// MARK: - Request DTOs (batch fallback retained for tests)

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
