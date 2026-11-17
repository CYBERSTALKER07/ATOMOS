import SwiftUI

struct LocationsView: View {
    @State private var vm = LocationsViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                Text("mobile_retailer.ui.primary_store_is_auto_created_from_your_shop_profile_add_branche")
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
            Section("Add branch") {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    .accessibilityLabel("Branch name")
                TextField("factory_portal.residual.text.address", text: $vm.address)
                    .accessibilityLabel("Branch delivery address")
                Button(vm.busy ? "…" : "Create") {
                    Task { await vm.create() }
                }
                .pegasusMinimumHitTarget()
                .disabled(!vm.canCreate)
                .accessibilityLabel("Create branch")
            }
            Section("Branches") {
                if vm.loading && vm.items.isEmpty {
                    ProgressView()
                }
                ForEach(vm.items) { loc in
                    VStack(alignment: .leading, spacing: 6) {
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(loc.name)
                                    .font(.headline)
                                if loc.isPrimary {
                                    Text("mobile_retailer.ui.primary")
                                        .font(.caption2)
                                        .foregroundStyle(AppTheme.textTertiary)
                                }
                                if vm.activeId == loc.locationId {
                                    Text("warehouse_portal.bins.text.active")
                                        .font(.caption2)
                                        .foregroundStyle(AppTheme.accent)
                                }
                            }
                            Text(loc.deliveryAddress ?? "No address")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(
                            "\(loc.name), \(loc.deliveryAddress ?? "No address")\(loc.isPrimary ? ", primary" : "")\(vm.activeId == loc.locationId ? ", active" : "")"
                        )

                        HStack(spacing: AppTheme.spacingSM) {
                            if loc.isActive && vm.activeId != loc.locationId {
                                Button("mobile_retailer.ui.use_for_checkout") {
                                    Task { await vm.switchTo(loc.locationId) }
                                }
                                .buttonStyle(.bordered)
                                .font(.caption)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busy)
                                .accessibilityLabel("Use \(loc.name) for checkout")
                            }
                            if loc.isActive && !loc.isPrimary {
                                Button("mobile_retailer.ui.set_primary") {
                                    Task { await vm.setPrimary(loc.locationId) }
                                }
                                .buttonStyle(.bordered)
                                .font(.caption)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busy)
                                .accessibilityLabel("Set \(loc.name) as primary branch")
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("retailer_desktop.settings.locations.text.locations")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
