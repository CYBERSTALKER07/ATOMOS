import SwiftUI

struct JoinRequestsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = JoinRequestsViewModel()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
                SupplierSectionHeader(
                    title: "supplier_portal.join_requests.inbox_title",
                    subtitle: "supplier_portal.join_requests.inbox_subtitle"
                )

                Picker("Status", selection: $vm.statusFilter) {
                    Text(LocalizedStringKey("supplier_portal.join_requests.status_pending")).tag("PENDING")
                    Text(LocalizedStringKey("supplier_portal.join_requests.status_approved")).tag("APPROVED")
                    Text(LocalizedStringKey("supplier_portal.join_requests.status_rejected")).tag("REJECTED")
                }
                .pickerStyle(.segmented)
                .onChange(of: vm.statusFilter) { _, _ in
                    Task { await vm.load() }
                }

                if let notice = vm.notice {
                    Text(notice)
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.success)
                        .padding(SupplierTheme.spacingSM)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(SupplierTheme.success.opacity(0.12), in: RoundedRectangle(cornerRadius: SupplierTheme.radiusSM))
                }

                if vm.loading && vm.requests.isEmpty {
                    SupplierLoadingView(title: "supplier_portal.join_requests.inbox_title")
                } else if let error = vm.error, vm.requests.isEmpty {
                    SupplierErrorView(message: error) {
                        Task { await vm.load() }
                    }
                } else if vm.requests.isEmpty {
                    SupplierEmptyView(
                        title: "supplier_portal.join_requests.inbox_title",
                        message: "supplier_portal.join_requests.empty_inbox"
                    )
                } else {
                    ForEach(vm.requests) { item in
                        requestCard(item)
                    }
                }
            }
            .padding(SupplierTheme.spacingLG)
            .supplierReadableWidth()
        }
        .background(SupplierTheme.background)
        .navigationTitle(LocalizedStringKey("supplier_portal.join_requests.inbox_title"))
        .refreshable { await vm.load() }
        .task { await vm.load() }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
    }

    private func requestCard(_ item: JoinRequestItem) -> some View {
        let reasonBinding = Binding<String>(
            get: { vm.rejectionReasonById[item.requestId] ?? "" },
            set: { vm.rejectionReasonById[item.requestId] = $0 }
        )
        let isBusy = vm.busyId == item.requestId

        return VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.requesterName?.isEmpty == false ? item.requesterName! : item.requesterPhone)
                        .font(.headline)
                    Text("\(item.requesterPhone) · INN \(item.normalizedInn)")
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(SupplierTheme.secondaryLabel)
                }
                Spacer()
                SupplierStatusBadge(text: item.status)
            }

            HStack {
                Text(LocalizedStringKey("supplier_portal.join_requests.role_sought"))
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
                Text(item.roleSought)
                    .font(.caption.weight(.semibold))
                Spacer()
                Text(item.createdAt)
                    .font(.caption2)
                    .foregroundStyle(SupplierTheme.tertiaryLabel)
            }

            if item.status.uppercased() == "PENDING" {
                TextField(
                    LocalizedStringKey("supplier_portal.join_requests.reject_reason_placeholder"),
                    text: reasonBinding
                )
                .textFieldStyle(.roundedBorder)

                HStack(spacing: SupplierTheme.spacingSM) {
                    Button {
                        Task { await vm.decide(requestId: item.requestId, approve: true) }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.join_requests.approve_button"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(SupplierTheme.primaryAction)
                    .foregroundStyle(SupplierTheme.onPrimaryAction)
                    .disabled(vm.busyId != nil)

                    Button(role: .destructive) {
                        Task { await vm.decide(requestId: item.requestId, approve: false) }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.join_requests.reject_button"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isBusy || vm.busyId != nil)
                }
            } else if let reason = item.decisionReason, !reason.isEmpty {
                Text(reason)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
        }
        .supplierCard()
    }
}
