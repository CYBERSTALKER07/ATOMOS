import SwiftUI

struct ClaimsView: View {
    @State private var claims: [SupplierClaim] = []
    @State private var statusFilter = "OPEN"
    @State private var settlementMode = "LEDGER_ONLY"
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?
    @State private var lastSettlement: String?

    private let settlementModes: [(String, String)] = [
        ("LEDGER_ONLY", "Ledger only"),
        ("STORE_CREDIT", "Store credit"),
        ("GATEWAY_REFUND", "Card refund (GP)"),
    ]

    var body: some View {
        List {
            Section {
                Picker("Status", selection: $statusFilter) {
                    Text("OPEN").tag("OPEN")
                    Text("UNDER_REVIEW").tag("UNDER_REVIEW")
                    Text("All").tag("")
                }
                .onChange(of: statusFilter) { _, _ in
                    Task { await load() }
                }

                Picker("Settlement", selection: $settlementMode) {
                    ForEach(settlementModes, id: \.0) { mode in
                        Text(mode.1).tag(mode.0)
                    }
                }

                if let lastSettlement {
                    Text("Last settle: \(lastSettlement)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                NavigationLink {
                    ClaimChargebacksView()
                } label: {
                    Label("Claim chargebacks ledger", systemImage: "list.bullet.rectangle")
                }
            }

            if loading {
                Section {
                    ProgressView("Loading claims…")
                }
            } else if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("Retry") { Task { await load() } }
                }
            } else if claims.isEmpty {
                Section {
                    Text("No claims in this filter. Retailer post-delivery claims appear within the 48h window.")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Claims") {
                    ForEach(claims) { claim in
                        VStack(alignment: .leading, spacing: 6) {
                            HStack {
                                Text(claim.claimType)
                                    .font(.caption.weight(.semibold))
                                Text(claim.status)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Text(claim.claimId)
                                .font(.caption.monospaced())
                                .foregroundStyle(.tint)
                            Text("Order \(claim.orderId) · Retailer \(claim.retailerId)")
                                .font(.caption)
                            Text("\(claim.amountMinor ?? 0) \(claim.currency ?? "UZS")")
                                .font(.subheadline.weight(.medium))
                            if let description = claim.description, !description.isEmpty {
                                Text(description).font(.caption).foregroundStyle(.secondary)
                            }
                            if claim.status == "OPEN" || claim.status == "UNDER_REVIEW" {
                                HStack {
                                    Button("Approve (\(settlementModes.first { $0.0 == settlementMode }?.1 ?? settlementMode))") {
                                        Task { await approve(claim.claimId) }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(busyId == claim.claimId)

                                    Button("Reject", role: .destructive) {
                                        Task { await reject(claim.claimId) }
                                    }
                                    .disabled(busyId == claim.claimId)
                                }
                                .padding(.top, 4)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .navigationTitle("Claims")
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            claims = try await SupplierOperationsService.listSupplierClaims(
                status: statusFilter.isEmpty ? nil : statusFilter
            )
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func approve(_ claimId: String) async {
        busyId = claimId
        lastSettlement = nil
        defer { busyId = nil }
        do {
            let resp = try await SupplierOperationsService.approveClaim(
                claimId: claimId,
                request: ApproveClaimRequest(
                    resolutionNote: "approved_via_supplier_ios",
                    settlementMode: settlementMode,
                    skipGatewayRefund: settlementMode != "GATEWAY_REFUND"
                )
            )
            if let s = resp.settlement {
                lastSettlement =
                    "\(s.mode) · \(s.amountMinor) · refund=\(s.gatewayRefunded) · id=\(s.chargebackId ?? "—")"
            }
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func reject(_ claimId: String) async {
        busyId = claimId
        defer { busyId = nil }
        do {
            _ = try await SupplierOperationsService.rejectClaim(
                claimId: claimId,
                request: RejectClaimRequest(resolutionNote: "rejected_via_supplier_ios")
            )
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
