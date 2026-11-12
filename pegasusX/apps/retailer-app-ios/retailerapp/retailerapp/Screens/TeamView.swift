import SwiftUI

/// Retail OS Phase 1 — team roster + invite.
struct TeamView: View {
    @State private var vm = TeamViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                Text("mobile_retailer.ui.invite_staff_with_roles_first_invite_enables_the_team_pack")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
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
                            Button("supplier_portal.demand.signals.text.deactivate", role: .destructive) {
                                Task { await vm.deactivate(m.userId) }
                            }
                            .font(.caption)
                            .pegasusMinimumHitTarget()
                            .disabled(vm.busy)
                            .accessibilityLabel("Deactivate \(m.name)")
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
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
