import SwiftUI

/// Retail OS Phase 1 — team roster + invite.
struct TeamView: View {
    @State private var vm = TeamViewModel()
    @State private var governanceVm = OwnerGovernanceViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                Text("mobile_retailer.ui.invite_staff_with_roles_first_invite_enables_the_team_pack")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                if governanceVm.isOwner {
                    NavigationLink(destination: OwnerApprovalsView()) {
                        Label(String(localized: "mobile_retailer.approval.inbox_title"), systemImage: "person.badge.shield.checkmark")
                            .font(.subheadline.weight(.semibold))
                    }
                }
            }
            if let actionBanner = governanceVm.actionBanner {
                Section {
                    Text(actionBanner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                }
            }
            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }
            if let errorText = vm.errorText {
                Section {
                    Text(errorText)
                        .foregroundStyle(AppTheme.destructive)
                    Button("common.action.retry") {
                        Task { await vm.load() }
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Retry loading team members")
                }
            }
            Section("Invite") {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    .accessibilityLabel("Staff member name")
                TextField("common.field.phone", text: $vm.phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                    .accessibilityLabel("Staff member phone number")
                SecureField("Password", text: $vm.password)
                    .accessibilityLabel("Staff member password")
                TextField("mobile_retailer.ui.role_e_g_cashier", text: $vm.role)
                    .accessibilityLabel("Staff role")
                Button(vm.busy ? "…" : "Create member") {
                    Task { await vm.invite() }
                }
                .pegasusMinimumHitTarget()
                .disabled(vm.busy || vm.name.isEmpty || vm.phone.isEmpty || vm.password.isEmpty)
                .accessibilityLabel("Create member")
            }
            if !vm.joinRequests.isEmpty {
                Section(String(localized: "retailer_desktop.tax_identity.join_requests_title")) {
                    ForEach(vm.joinRequests) { req in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(req.requesterName?.isEmpty == false ? (req.requesterName ?? req.requesterPhone) : req.requesterPhone)
                                .font(.headline)
                            Text("\(req.requesterPhone) • \(req.roleSought)")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textSecondary)
                            HStack(spacing: 12) {
                                Button(String(localized: "retailer_desktop.tax_identity.approve_join")) {
                                    Task { _ = await vm.approveJoinRequest(requestId: req.requestId) }
                                }
                                .buttonStyle(.borderedProminent)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.decidingRequestId == req.requestId)
                                .accessibilityLabel("Approve join request from \(req.requesterPhone)")

                                Button(String(localized: "retailer_desktop.tax_identity.reject_join"), role: .destructive) {
                                    Task { _ = await vm.rejectJoinRequest(requestId: req.requestId) }
                                }
                                .buttonStyle(.bordered)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.decidingRequestId == req.requestId)
                                .accessibilityLabel("Reject join request from \(req.requesterPhone)")
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
            Section("Roster") {
                if vm.loading && vm.members.isEmpty {
                    ProgressView()
                }
                ForEach(vm.members) { m in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(m.name).font(.headline)
                            if m.isOwner {
                                Text("mobile_retailer.ui.owner").font(.caption2).foregroundStyle(AppTheme.textTertiary)
                            }
                            if !m.isActive {
                                Text("retailer_desktop.settings.team.text.inactive").font(.caption2).foregroundStyle(.red)
                            }
                        }
                        Text(L10n.format("mobile_retailer.ui.phone_retailerrole", "\(m.phone)", "\(m.retailerRole)"))
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                        if !m.isOwner && m.isActive {
                            HStack(spacing: 12) {
                                if governanceVm.isOwner {
                                    Button(String(localized: "mobile_retailer.mandate.title")) {
                                        Task { await governanceVm.openMandateEditor(userId: m.userId, memberName: m.name) }
                                    }
                                    .font(.caption.weight(.semibold))
                                    .buttonStyle(.bordered)
                                    .pegasusMinimumHitTarget()
                                    .disabled(vm.busy)
                                    .accessibilityLabel("Edit seller mandate for \(m.name)")
                                }
                                Button("supplier_portal.demand.signals.text.deactivate", role: .destructive) {
                                    Task { await vm.deactivate(m.userId) }
                                }
                                .font(.caption)
                                .buttonStyle(.borderless)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busy)
                                .accessibilityLabel("Deactivate \(m.name)")
                            }
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(m.name), role \(m.retailerRole), phone \(m.phone)\(m.isActive ? ", active" : ", inactive")")
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("retailer_desktop.settings.team.text.team")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await vm.load()
            await governanceVm.loadRoleAndApprovals()
        }
        .refreshable {
            await vm.load()
            await governanceVm.loadRoleAndApprovals()
        }
        .sheet(isPresented: Binding(
            get: { governanceVm.mandateForm != nil },
            set: { if !$0 { governanceVm.mandateForm = nil } }
        )) {
            SellerMandateEditorSheet(vm: governanceVm)
        }
    }
}
