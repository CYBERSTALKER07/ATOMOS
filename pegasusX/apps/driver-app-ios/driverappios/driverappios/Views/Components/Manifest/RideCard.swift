import SwiftUI
import CoreLocation

struct RideCard: View {
    let mission: Mission
    let index: Int
    let loadSeqLabel: String?
    let location: CLLocation?
    let onSelect: () -> Void
    
    var body: some View {
        Button {
            onSelect()
        } label: {
            VStack(alignment: .leading, spacing: 16) {
                // Loading sequence badge
                if let label = loadSeqLabel {
                    Text(label)
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.buttonFg)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(LabTheme.fg, in: RoundedRectangle(cornerRadius: 4))
                }
                
                // Top row: order id + status
                HStack {
                    Text(mission.order_id)
                        .font(.system(size: 15, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    
                    Spacer()
                    
                    StatusPill(label: mission.state, color: LabTheme.fg)
                }
                
                // Amount row
                HStack(spacing: 12) {
                    InfoChip(icon: "creditcard", text: mission.gateway)
                    InfoChip(icon: "banknote", text: mission.amount.formattedAmount)
                }
                
                // Coordinates
                VStack(alignment: .leading, spacing: 6) {
                    Text("DELIVERY TARGET")
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                    
                    Text(String(format: "%.4f, %.4f", mission.target_lat, mission.target_lng))
                        .font(.system(size: 14, weight: .semibold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                }
                
                // Distance + action
                HStack {
                    DistanceInfo(mission: mission, location: location)
                    Spacer()
                    Image(systemName: "arrow.right")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                        .padding(8)
                        .background(LabTheme.fg.opacity(0.08), in: Circle())
                }
                
                // Bottom accent bar
                RoundedRectangle(cornerRadius: 2)
                    .fill(LabTheme.fg.opacity(0.12))
                    .frame(height: 2)
            }
            .padding(LabTheme.s20)
            .labCard()
        }
        .buttonStyle(.pressable)
        .staggeredAppear(index: index)
    }
}
