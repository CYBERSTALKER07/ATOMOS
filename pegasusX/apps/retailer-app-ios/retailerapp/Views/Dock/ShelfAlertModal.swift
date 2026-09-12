import SwiftUI

struct ShelfAlertModal: View {
    let productId: String
    let currentStock: Int64
    let onDismiss: () -> Void

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 48))
                .foregroundColor(.red)
            
            Text("RESTOCK NEEDED")
                .font(.title2)
                .bold()
                .foregroundColor(Color(red: 0.6, green: 0.1, blue: 0.1)) // Dark red
            
            Text("Product: \(productId) has dropped to \(currentStock) items on the floor.")
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundColor(Color(red: 0.5, green: 0.1, blue: 0.1))
            
            Button(action: onDismiss) {
                Text("ACKNOWLEDGE")
                    .font(.headline)
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.red)
                    .cornerRadius(8)
            }
            .padding(.top, 8)
        }
        .padding(24)
        .background(Color(red: 1.0, green: 0.9, blue: 0.9)) // Light red background
        .cornerRadius(16)
        .shadow(radius: 10)
        .padding(24)
    }
}
