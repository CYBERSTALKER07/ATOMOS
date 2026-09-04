import Foundation

/// Persists retailer profile updates against `/v1/retailer/profile`.
enum RetailerProfileService {
    static func saveProfile(
        api: APIClient,
        retailerId: String,
        name: String?,
        company: String?,
        phone: String?,
        location: String? = nil,
        regionId: String? = nil,
        receivingWindowOpen: String? = nil,
        receivingWindowClose: String? = nil
    ) async throws -> RetailerProfileResponse {
        let request = RetailerProfileRequest(
            name: name,
            company: company,
            phone: phone,
            location: location,
            regionId: regionId,
            receivingWindowOpen: receivingWindowOpen,
            receivingWindowClose: receivingWindowClose
        )
        let fingerprint = profileFingerprint(
            name: name,
            company: company,
            phone: phone,
            location: location,
            regionId: regionId,
            receivingWindowOpen: receivingWindowOpen,
            receivingWindowClose: receivingWindowClose
        )
        try await api.updateProfile(
            request: request,
            idempotencyKey: RetailerIdempotency.profileUpdate(
                retailerId: retailerId,
                payloadFingerprint: fingerprint
            )
        )
        return try await api.getProfile()
    }

    private static func profileFingerprint(
        name: String?,
        company: String?,
        phone: String?,
        location: String?,
        regionId: String?,
        receivingWindowOpen: String?,
        receivingWindowClose: String?
    ) -> String {
        let fields: [(String, String)] = [
            ("company", company ?? ""),
            ("location", location ?? ""),
            ("name", name ?? ""),
            ("phone", phone ?? ""),
            ("receiving_window_close", receivingWindowClose ?? ""),
            ("receiving_window_open", receivingWindowOpen ?? ""),
            ("region_id", regionId ?? ""),
        ]
        return fields
            .filter { !$0.1.isEmpty }
            .map { "\($0.0)=\($0.1)" }
            .joined(separator: "|")
    }
}
