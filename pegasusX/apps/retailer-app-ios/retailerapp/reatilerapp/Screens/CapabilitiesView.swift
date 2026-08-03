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
                Text("Solo shops run on Core alone. Enable packs as you grow — hard dependencies are enforced.")
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
                    Button("Retry") { Task { await load() } }
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
                                Text("Always on")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textTertiary)
                            } else if pack.enabled {
                                Button("Disable") {
                                    Task { await disable(pack.id) }
                                }
                                .disabled(busyId == pack.id)
                            } else {
                                Button("Enable") {
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
                            Text("Requires: \(hard.joined(separator: ", "))")
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("Store capabilities")
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
