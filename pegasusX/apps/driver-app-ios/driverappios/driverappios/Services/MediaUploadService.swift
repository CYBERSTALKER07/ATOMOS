//
//  MediaUploadService.swift
//  driverappios
//
//  Signed GCS PUT for OS&D / damage evidence photos.
//

import Foundation
import UIKit

enum MediaUploadService {
    struct Ticket: Decodable {
        let uploadURL: String
        let publicURL: String
        let contentType: String?

        enum CodingKeys: String, CodingKey {
            case uploadURL = "upload_url"
            case publicURL = "public_url"
            case imageURL = "image_url"
            case contentType = "content_type"
        }

        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            self.uploadURL = try c.decode(String.self, forKey: .uploadURL)
            if let pub = try c.decodeIfPresent(String.self, forKey: .publicURL) {
                self.publicURL = pub
            } else {
                self.publicURL = try c.decode(String.self, forKey: .imageURL)
            }
            self.contentType = try c.decodeIfPresent(String.self, forKey: .contentType)
        }
    }

    /// Uploads a JPEG and returns the public URL to attach as photo_url on exception reports.
    static func uploadJPEG(
        image: UIImage,
        purpose: String = "driver_exception",
        orderId: String?,
        api: APIClient = .shared
    ) async throws -> String {
        guard let data = image.jpegData(compressionQuality: 0.82) else {
            throw APIError.decodingError
        }
        var path = "v1/media/upload-ticket?purpose=\(purpose)&ext=jpg"
        if let orderId, !orderId.isEmpty {
            let enc = orderId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? orderId
            path += "&order_id=\(enc)"
        }
        let ticket: Ticket = try await api.get(path)
        guard let putURL = URL(string: ticket.uploadURL) else {
            throw APIError.invalidURL
        }
        var req = URLRequest(url: putURL)
        req.httpMethod = "PUT"
        req.setValue(ticket.contentType ?? "image/jpeg", forHTTPHeaderField: "Content-Type")
        req.httpBody = data
        let (_, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            throw APIError.httpError((response as? HTTPURLResponse)?.statusCode ?? 0)
        }
        return ticket.publicURL
    }
}
