import SwiftUI

struct RescueAcceptanceDialog: View {
    let targetDriverId: String
    let ordersToRescue: Int
    let location: String
    let onAccept: () -> Void
    let onDecline: () -> Void

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "lifepreserver.fill")
                .font(.system(size: 48))
                .foregroundColor(.orange)
            
            Text("URGENT RESCUE REQUEST")
                .font(.title3)
                .bold()
                .foregroundColor(.red)
            
            VStack(spacing: 8) {
                Text("Driver \(targetDriverId.prefix(8)) is stalled at:")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Text(location)
                    .font(.headline)
                Text("They have \(ordersToRescue) orders at risk.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            .multilineTextAlignment(.center)
            
            HStack(spacing: 16) {
                Button(action: onDecline) {
                    Text("DECLINE")
                        .font(.headline)
                        .foregroundColor(.gray)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.gray.opacity(0.2))
                        .cornerRadius(8)
                }
                
                Button(action: onAccept) {
                    Text("ACCEPT RESCUE")
                        .font(.headline)
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.orange)
                        .cornerRadius(8)
                }
            }
        }
        .padding(24)
        .background(Color.white)
        .cornerRadius(16)
        .shadow(radius: 10)
        .padding(24)
    }
}
