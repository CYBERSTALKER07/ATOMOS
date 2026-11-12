import Foundation
import Observation
import PhotosUI
import SwiftUI
import UIKit

@Observable
@MainActor
final class ShopClosedBannerViewModel {
    var isSubmitting = false
    var errorText: String?
    var bypassPending = false
    var bypassPhotoURL: String?
    var uploading = false
    var photoItem: PhotosPickerItem?

    private let api = APIClient.shared

    func labelForOption(_ option: String) -> String {
        switch option {
        case "OPEN_NOW": return "I am open now"
        case "5_MIN": return "I will be back in 5 mins"
        case "CALL_ME": return "Call me"
        case "CLOSED_TODAY": return "Closed for the day"
        case "RESCHEDULE": return "Reschedule delivery"
        case "CREDIT_LEAVE": return "Leave on credit"
        case "CANCEL": return "Cancel remaining"
        case "AUTHORIZE_BYPASS": return "Authorize bypass offload"
        default: return option
        }
    }

    func handleReconnect() {
        if isSubmitting {
            isSubmitting = false
            errorText = "Connection restored — verify response before retrying."
        }
    }

    func cancelBypass() {
        Haptics.light()
        bypassPending = false
        bypassPhotoURL = nil
        photoItem = nil
        errorText = nil
    }

    func selectOption(_ option: String, orderId: String, onRespond: @escaping (String) -> Void) {
        if option == "AUTHORIZE_BYPASS" {
            Haptics.selection()
            bypassPending = true
            errorText = nil
            return
        }
        respond(with: option, orderId: orderId, onRespond: onRespond)
    }

    func uploadBypassPhoto(_ item: PhotosPickerItem, orderId: String) async {
        uploading = true
        errorText = nil
        defer { uploading = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else {
                Haptics.error()
                errorText = "Could not read photo"
                return
            }
            bypassPhotoURL = try await MediaUploadService.uploadJPEG(
                image: image,
                purpose: "claim_evidence",
                orderId: orderId,
                api: api
            )
            Haptics.success()
        } catch {
            Haptics.error()
            bypassPhotoURL = nil
            errorText = error.localizedDescription
        }
    }

    func respond(
        with option: String,
        orderId: String,
        photoURL: String? = nil,
        onRespond: @escaping (String) -> Void
    ) {
        if option == "AUTHORIZE_BYPASS", (photoURL ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            Haptics.warning()
            bypassPending = true
            errorText = "Doorway / drop-off photo is required to authorize bypass."
            return
        }
        isSubmitting = true
        errorText = nil
        Haptics.medium()
        Task {
            do {
                try await api.respondToShopClosed(
                    orderId: orderId,
                    response: option,
                    photoURL: photoURL
                )
                Haptics.success()
                onRespond(option)
            } catch {
                Haptics.error()
                let raw = error.localizedDescription
                if raw.contains("photo_url_required_for_bypass") {
                    errorText = "Doorway / drop-off photo is required to authorize bypass."
                    bypassPending = true
                } else {
                    errorText = RetailerErrorSupport.message(
                        for: error,
                        restricted: "Shop status confirmation is restricted for this account.",
                        offline: "Offline mode active. Reconnect and retry your response.",
                        fallback: "Could not submit response. Please try again."
                    )
                }
                isSubmitting = false
            }
        }
    }
}
