import SwiftUI

/// Tactical KPI header row — mirrors Android `ManifestKpiGrid` and pegasus `ManifestWorkflow` cards.
struct ManifestKpiGrid: View {
    let manifest: Manifest

    private var progress: Double {
        let cap = max(manifest.maxVolumeVu ?? 0, 0.001)
        let total = manifest.totalVolumeVu ?? 0
        return min(max(total / cap, 0), 1)
    }

    private var volumeLabel: String {
        String(format: "%.1f / %.1f VU", manifest.totalVolumeVu ?? 0, manifest.maxVolumeVu ?? 0)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: TermTheme.s20) {
            HStack(alignment: .top, spacing: TermTheme.s20) {
                PayloadStatusBadge(text: manifest.state, large: true)
                KpiTile(label: "PAYLOAD VOLUME", value: volumeLabel) {
                    ProgressView(value: progress)
                        .tint(TermTheme.accent)
                        .scaleEffect(x: 1, y: 1.5, anchor: .center)
                }
            }

            HStack(spacing: TermTheme.s20) {
                KpiTile(
                    label: "TARGET STOPS",
                    value: "\(manifest.stopCount ?? 0) UNITS"
                )
                if let region = manifest.regionCode, !region.isEmpty {
                    KpiTile(
                        label: "DEPLOYMENT ZONE",
                        value: region.uppercased()
                    )
                }
            }

            Text(L10n.format("mobile_payload.ui.manifest_prefix", "\(manifest.manifestId.prefix(8))"))
                .font(.system(size: 11, weight: .bold, design: .monospaced))
                .foregroundStyle(TermTheme.tertiary)
        }
    }
}
