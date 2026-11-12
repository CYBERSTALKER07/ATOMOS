import SwiftUI
import UIKit

struct RescueAcceptanceDialog: View {
    let targetDriverId: String
    let ordersToRescue: Int
    let location: String
    let onAccept: () -> Void
    let onDecline: () -> Void

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "lifepreserver.fill")
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(.orange)
                .accessibilityHidden(true)

            Text("URGENT RESCUE REQUEST")
                .font(.title3)
                .bold()
                .foregroundStyle(.red)

            VStack(spacing: 8) {
                Text("Driver \(targetDriverId.prefix(8)) is stalled at:")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Text(location)
                    .font(.headline)
                Text("They have \(ordersToRescue) orders at risk.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .multilineTextAlignment(.center)
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Urgent rescue request. Driver \(targetDriverId.prefix(8)) stalled at \(location) with \(ordersToRescue) orders at risk.")

            HStack(spacing: 16) {
                Button {
                    UIImpactFeedbackGenerator(style: .light).impactOccurred()
                    onDecline()
                } label: {
                    Text("DECLINE")
                        .font(.headline)
                        .foregroundStyle(.gray)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 8)
                        .background(Color.gray.opacity(0.2))
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                .accessibilityLabel("Decline rescue request")

                Button {
                    UINotificationFeedbackGenerator().notificationOccurred(.success)
                    onAccept()
                } label: {
                    Text("ACCEPT RESCUE")
                        .font(.headline)
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 8)
                        .background(Color.orange)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                .accessibilityLabel("Accept rescue request")
            }
        }
        .padding(24)
        .background(Color.white)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .shadow(radius: 10)
        .padding(24)
    }
}
