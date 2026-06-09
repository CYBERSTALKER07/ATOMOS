import Foundation

/// Persists retailer profile updates against `/v1/retailer/profile`.
enum RetailerProfileService {
    static func saveProfile(
        api: APIClient,
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
        try await api.updateProfile(request: request)
        return try await api.getProfile()
    }
}
