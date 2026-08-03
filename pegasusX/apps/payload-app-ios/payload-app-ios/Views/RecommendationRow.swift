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
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                    
                    let meta = [rec.licensePlate, rec.vehicleClass, rec.truckStatus]
                        .compactMap { ($0?.isEmpty == false) ? $0 : nil }
                        .joined(separator: " • ")
                        .uppercased()
                    
                    if !meta.isEmpty {
                        Text(meta)
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                    }
                }
                
                Spacer()
                
                VStack(alignment: .trailing, spacing: 4) {
                    Text("OPTIMIZATION_SCORE")
                        .font(.system(size: 8, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.2f", rec.score ?? 0))
                        .font(.system(size: 16, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.live)
                }
            }
            
            HStack(spacing: 16) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("EST_TRAVEL")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f KM", rec.distanceKm ?? 0))
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                VStack(alignment: .leading, spacing: 2) {
                    Text("FREE_CAP")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f VU", rec.freeVolumeVu ?? 0))
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                         Spacer()
            }
            
            HStack(spacing: 8) {
                Button(action: onPickComplete) {
                    Text("COMPLETE")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                        .background(TermTheme.accent.opacity(0.1))
                        .foregroundStyle(TermTheme.accent)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                
                Button(action: onPickPartial) {
                    Text("PARTIAL")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                        .background(TermTheme.secondary.opacity(0.1))
                        .foregroundStyle(TermTheme.secondary)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
            }
            .padding(.top, 4)
        }
        .padding(16)
        .background(TermTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .tacticalCard()
    }
}
