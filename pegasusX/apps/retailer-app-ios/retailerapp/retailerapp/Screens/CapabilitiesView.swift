import SwiftUI

/// Retail OS Phase 0 — store capability packs (Settings → Store capabilities).
struct CapabilitiesView: View {
    @State private var vm = CapabilitiesViewModel()

    var body: some View {
        List {
            Section {
                Text("mobile_retailer.ui.solo_shops_run_on_core_alone_enable_packs_as_you_grow_hard_depen")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }
            if vm.loading && vm.packs.isEmpty {
                Section { ProgressView("Loading…") }
            }
            if let errorText = vm.errorText {
                Section {
                    Text(errorText)
                        .foregroundStyle(AppTheme.destructive)
                    Button("common.action.retry") {
                        Task { await vm.load() }
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Retry loading capabilities")
                }
            }
            Section("Packs") {
                ForEach(vm.packs) { pack in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(pack.name)
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                Text(pack.id)
                                    .font(.system(.caption2, design: .monospaced))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(pack.name), pack ID \(pack.id), \(pack.enabled ? "enabled" : "disabled")")

                            Spacer()
                            if pack.alwaysOn == true {
                                Text("retailer_desktop.settings.capabilities.text.always_on")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textTertiary)
                            } else if pack.enabled {
                                Button("mobile_retailer.ui.disable") {
                                    Task { await vm.disable(pack.id) }
                                }
                                .buttonStyle(.bordered)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busyId == pack.id)
                                .accessibilityLabel("Disable \(pack.name)")
                            } else {
                                Button("mobile_retailer.ui.enable") {
                                    Task { await vm.enable(pack.id) }
                                }
                                .buttonStyle(.borderedProminent)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busyId == pack.id)
                                .accessibilityLabel("Enable \(pack.name)")
                            }
                        }
                        Text(pack.description)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                        if let hard = pack.hardDeps, !hard.isEmpty {
                            Text(L10n.format("mobile_retailer.ui.requires_joined", "\(hard.joined(separator: ", "))"))
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("retailer_desktop.settings.capabilities.text.store_capabilities")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
