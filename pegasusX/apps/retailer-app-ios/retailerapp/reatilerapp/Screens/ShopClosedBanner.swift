import PhotosUI
import SwiftUI
import UIKit

struct ShopClosedBanner: View {
    let event: ShopClosedAlertEvent
    let onRespond: (String) -> Void
    @State private var isSubmitting = false
    @State private var errorText: String? = nil
    @State private var socket = RetailerWebSocket.shared
    @State private var bypassPending = false
    @State private var bypassPhotoURL: String? = nil
    @State private var uploading = false
    @State private var photoItem: PhotosPickerItem? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundColor(.yellow)
                Text(L10n.format("mobile_retailer.ui.driver_drivername_reported_your_shop_is_closed", "\(event.driverName)"))
                    .font(.headline)
            }

            Text("mobile_retailer.ui.please_confirm_your_status_so_we_can_manage_your_order")
                .font(.subheadline)
                .foregroundColor(.secondary)

            if let errorText = errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundColor(.red)
            }

            if bypassPending {
                Text("mobile_retailer.ui.doorway_drop_off_proof_is_required_for_authorize_bypass")
                    .font(.caption)
                    .foregroundColor(.secondary)
                PhotosPicker(selection: $photoItem, matching: .images) {
                    Label(
                        uploading ? "Uploading…" : (bypassPhotoURL != nil ? "Replace photo" : "Take or choose photo"),
                        systemImage: "camera"
                    )
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.secondary.opacity(0.15))
                    .cornerRadius(8)
                }
                .disabled(isSubmitting || uploading)
                .onChange(of: photoItem) { _, item in
                    guard let item else { return }
                    Task { await uploadBypassPhoto(item) }
                }
                if bypassPhotoURL != nil {
                    Button("mobile_retailer.ui.confirm_bypass") {
                        respond(with: "AUTHORIZE_BYPASS", photoURL: bypassPhotoURL)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(isSubmitting || uploading)
                }
                Button("mobile_retailer.ui.cancel_bypass") {
                    bypassPending = false
                    bypassPhotoURL = nil
                    photoItem = nil
                    errorText = nil
                }
                .disabled(isSubmitting)
            }

            VStack(spacing: 8) {
                ForEach(event.options, id: \.self) { option in
                    Button(action: {
                        if option == "AUTHORIZE_BYPASS" {
                            bypassPending = true
                            errorText = nil
                            return
                        }
                        respond(with: option)
                    }) {
                        Text(labelForOption(option))
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                    }
                    .disabled(isSubmitting || uploading)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(radius: 5)
        .padding()
        .onChange(of: socket.reconnectEpoch) { _, _ in
            if isSubmitting {
                isSubmitting = false
                errorText = "Connection restored — verify response before retrying."
            }
        }
    }

    private func labelForOption(_ option: String) -> String {
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

    private func uploadBypassPhoto(_ item: PhotosPickerItem) async {
        uploading = true
        errorText = nil
        defer { uploading = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else {
                errorText = "Could not read photo"
                return
            }
            bypassPhotoURL = try await MediaUploadService.uploadJPEG(
                image: image,
                purpose: "claim_evidence",
                orderId: event.orderId
            )
        } catch {
            bypassPhotoURL = nil
            errorText = error.localizedDescription
        }
    }

    private func respond(with option: String, photoURL: String? = nil) {
        if option == "AUTHORIZE_BYPASS", (photoURL ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            bypassPending = true
            errorText = "Doorway / drop-off photo is required to authorize bypass."
            return
        }
        isSubmitting = true
        errorText = nil
        Task {
            do {
                try await APIClient.shared.respondToShopClosed(
                    orderId: event.orderId,
                    response: option,
                    photoURL: photoURL
                )
                onRespond(option)
            } catch {
                let raw = error.localizedDescription
                if raw.contains("photo_url_required_for_bypass") {
                    errorText = "Doorway / drop-off photo is required to authorize bypass."
                    bypassPending = true
                } else {
                    errorText = RetailerErrorSupport.message(
                        for: error,
                        restricted: "Shop status confirmation is restricted for this account.",
                        offline: "Offline mode active. Reconnect and retry your response.",
                        fallback: "Could not submit response. Please try again.",
                    )
                }
                isSubmitting = false
            }
        }
    }
}
