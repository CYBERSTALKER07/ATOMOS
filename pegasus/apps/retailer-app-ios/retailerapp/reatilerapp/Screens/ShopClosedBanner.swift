import SwiftUI

struct ShopClosedBanner: View {
    let event: ShopClosedAlertEvent
    let onRespond: (String) -> Void
    @State private var isSubmitting = false
    @State private var errorText: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundColor(.yellow)
                Text("Driver \(event.driverName) reported your shop is closed.")
                    .font(.headline)
            }
            
            Text("Please confirm your status so we can manage your order.")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            if let errorText = errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundColor(.red)
            }
            
            VStack(spacing: 8) {
                ForEach(event.options, id: \.self) { option in
                    Button(action: {
                        respond(with: option)
                    }) {
                        Text(labelForOption(option))
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                    }
                    .disabled(isSubmitting)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(radius: 5)
        .padding()
    }
    
    private func labelForOption(_ option: String) -> String {
        switch option {
        case "OPEN_NOW": return "I am open now"
        case "5_MIN": return "I will be back in 5 mins"
        case "CALL_ME": return "Call me"
        case "CLOSED_TODAY": return "Closed for the day"
        default: return option
        }
    }
    
    private func respond(with option: String) {
        isSubmitting = true
        errorText = nil
        Task {
            do {
                try await APIClient.shared.respondToShopClosed(orderId: event.orderId, response: option)
                onRespond(option)
            } catch {
                errorText = RetailerErrorSupport.message(
                    for: error,
                    restricted: "Shop status confirmation is restricted for this account.",
                    offline: "Offline mode active. Reconnect and retry your response.",
                    fallback: "Could not submit response. Please try again.",
                )
                isSubmitting = false
            }
        }
    }
}
