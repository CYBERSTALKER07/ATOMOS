import CoreLocation
import SwiftUI

struct DistanceInfo: View {
    let mission: Mission
    let location: CLLocation?
    
    var body: some View {
        if let loc = location {
            let target = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
            let dist = haversineDistance(from: loc, to: target)
            let inRange = dist <= 500_000
            
            HStack(spacing: 6) {
                Circle()
                    .fill(inRange ? LabTheme.success : LabTheme.fgTertiary)
                    .frame(width: 6, height: 6)
                
                Text(formattedDistance(dist))
                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                    .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
            }
        }
    }
}
