import SwiftUI

struct PaymentConfigView: View {
    @State private var gateways: [PaymentGateway] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else if gateways.isEmpty {
                ContentUnavailableView("No Gateways", systemImage: "creditcard", description: Text("No payment gateways configured"))
            } else {
                List(gateways) { gw in
                    HStack {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(gw.name)
                                .font(.headline)
                            Text(gw.provider)
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        Image(systemName: gw.isActive ? "checkmark.circle.fill" : "xmark.circle")
                            .foregroundStyle(gw.isActive ? .green : .secondary)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.paymentConfig()
                gateways = resp.gateways
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
