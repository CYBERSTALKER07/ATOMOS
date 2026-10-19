import SwiftUI

/// Retail OS Phase 0 — store capability packs (Settings → Store capabilities).
struct CapabilitiesView: View {
    @State private var packs: [RetailerCapabilityPack] = []
    @State private var loading = true
    @State private var errorText: String?
    @State private var banner: String?
    @State private var busyId: String?

    private let api = APIClient.shared

    var body: some View {
        List {
            Section {
                Text("mobile_retailer.ui.solo_shops_run_on_core_alone_enable_packs_as_you_grow_hard_depen")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section {
                    Text(banner)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.accent)
                }
            }
            if loading && packs.isEmpty {
                Section { ProgressView("Loading…") }
            }
            if let errorText {
                Section {
                    Text(errorText).foregroundStyle(.red)
                    Button("common.action.retry") { Task { await load() } }
                }
            }
            Section("Packs") {
                ForEach(packs) { pack in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(pack.name)
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                Text(pack.id)
                                    .font(.system(.caption2, design: .monospaced))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            Spacer()
                            if pack.alwaysOn == true {
                                Text("retailer_desktop.settings.capabilities.text.always_on")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textTertiary)
                            } else if pack.enabled {
                                Button("mobile_retailer.ui.disable") {
                                    Task { await disable(pack.id) }
                                }
                                .disabled(busyId == pack.id)
                            } else {
                                Button("mobile_retailer.ui.enable") {
                                    Task { await enable(pack.id) }
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(busyId == pack.id)
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
        .navigationTitle("retailer_desktop.settings.capabilities.text.store_capabilities")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        loading = true
        errorText = nil
        defer { loading = false }
        do {
            let res = try await api.getCapabilities()
            packs = res.packs
        } catch {
            errorText = error.localizedDescription
        }
    }

    private func enable(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            let res = try await api.enableCapability(packId: id, acceptSoft: true, enableDeps: true)
            banner = res.message ?? "\(id) enabled"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func disable(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            _ = try await api.disableCapability(packId: id)
            banner = "\(id) disabled"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }
}
