import SwiftUI

struct ClaimsView: View {
    private let statusFilters = ["OPEN", "UNDER_REVIEW", "RESOLVED", "REJECTED", ""]

    @State private var status = "OPEN"
    @State private var claims: [WarehouseClaim] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack {
                        ForEach(statusFilters, id: \.self) { s in
                            Button(s.isEmpty ? "ALL" : s) { status = s }
                                .buttonStyle(.bordered)
                                .tint(status == s ? .accentColor : .secondary)
                        }
                    }
                    .padding(.horizontal, LabTheme.spacingMD)
                    .padding(.vertical, LabTheme.spacingSM)
                }
                HStack(spacing: LabTheme.spacingMD) {
                    NavigationLink("Returns inbound") { ReturnsView() }
                    NavigationLink("Exception triage") { ExceptionsView() }
                }
                .font(.subheadline)
                .padding(.horizontal, LabTheme.spacingMD)
                Text("Read-only prep queue. Approve/reject stays supplier HQ.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, LabTheme.spacingMD)
                    .padding(.bottom, LabTheme.spacingSM)

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
                            Button("Retry") { Task { await load() } }
                        }
                    } else if claims.isEmpty {
                        ContentUnavailableView("No claims in this filter", systemImage: "doc.text")
                    } else {
                        List(claims) { c in
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text("\(c.claimType) · \(c.status)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                Text(c.claimId)
                                    .font(.headline.monospaced())
                                if !c.orderId.isEmpty {
                                    NavigationLink("Order \(c.orderId)") {
                                        OrderDetailView(orderId: c.orderId)
                                    }
                                }
                                Text("Retailer \(c.retailerId) · \(c.amountMinor) \(c.currency)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                ForEach(c.lineItems) { li in
                                    Text("\(li.sku) × \(li.quantity)" + (li.reason.isEmpty ? "" : " (\(li.reason))"))
                                        .font(.caption)
                                }
                                if !c.description.isEmpty {
                                    Text(c.description)
                                        .font(.subheadline)
                                }
                            }
                            .padding(.vertical, 4)
                        }
                        .listStyle(.plain)
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Claims")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { Task { await load() } }
                }
            }
            .task(id: status) { await load() }
            .refreshable { await load() }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await WarehouseService.supplierClaims(
                status: status.isEmpty ? nil : status,
                limit: 50
            )
            claims = resp.claims
        } catch {
            self.error = error.localizedDescription
        }
    }
}
