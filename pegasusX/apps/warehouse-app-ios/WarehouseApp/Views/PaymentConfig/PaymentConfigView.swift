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
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if gateways.isEmpty {
                ContentUnavailableView("No Gateways", systemImage: "creditcard", description: Text("warehouse_portal.payment_config.text.no_payment_gateways_configured"))
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(gateways) { gw in
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
            }
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
