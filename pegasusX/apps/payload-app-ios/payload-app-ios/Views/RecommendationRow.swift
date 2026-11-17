import SwiftUI

struct RecommendationRow: View {
    let rec: TruckRecommendation
    let onPickComplete: () -> Void
    let onPickPartial: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(rec.driverName?.uppercased() ?? "ID-\(rec.driverId.prefix(8).uppercased())")
                        .font(.system(.subheadline, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.accent)

                    let meta = [rec.licensePlate, rec.vehicleClass, rec.truckStatus]
                        .compactMap { ($0?.isEmpty == false) ? $0 : nil }
                        .joined(separator: " • ")
                        .uppercased()

                    if !meta.isEmpty {
                        Text(meta)
                            .font(.system(.caption2, design: .monospaced, weight: .bold))
                            .foregroundStyle(TermTheme.secondary)
                    }
                }

                Spacer()

                VStack(alignment: .trailing, spacing: 4) {
                    Text("OPTIMIZATION_SCORE")
                        .font(.system(.caption2, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.2f", rec.score ?? 0))
                        .font(.system(.callout, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.live)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Driver \(rec.driverName ?? rec.driverId), score \(String(format: "%.2f", rec.score ?? 0)), distance \(String(format: "%.1f", rec.distanceKm ?? 0)) km, free capacity \(String(format: "%.1f", rec.freeVolumeVu ?? 0)) volume units")

            HStack(spacing: 16) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("EST_TRAVEL")
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f KM", rec.distanceKm ?? 0))
                        .font(.system(.caption, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.secondary)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text("FREE_CAP")
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f VU", rec.freeVolumeVu ?? 0))
                        .font(.system(.caption, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.secondary)
                }
                Spacer()
            }
            .accessibilityHidden(true)

            HStack(spacing: 8) {
                Button {
                    PegasusHaptics.medium()
                    onPickComplete()
                } label: {
                    Text("COMPLETE")
                        .font(.system(.caption, design: .monospaced, weight: .black))
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 8)
                        .background(TermTheme.accent.opacity(0.1))
                        .foregroundStyle(TermTheme.accent)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                .buttonStyle(.tactical)
                .accessibilityLabel("Reassign complete order to \(rec.driverName ?? rec.driverId)")

                Button {
                    PegasusHaptics.light()
                    onPickPartial()
                } label: {
                    Text("PARTIAL")
                        .font(.system(.caption, design: .monospaced, weight: .black))
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 8)
                        .background(TermTheme.secondary.opacity(0.1))
                        .foregroundStyle(TermTheme.secondary)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                .buttonStyle(.tactical)
                .accessibilityLabel("Reassign partial order to \(rec.driverName ?? rec.driverId)")
            }
            .padding(.top, 4)
        }
        .padding(16)
        .background(TermTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .tacticalCard()
    }
}
