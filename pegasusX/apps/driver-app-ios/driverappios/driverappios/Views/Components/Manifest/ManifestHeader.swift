import SwiftUI

struct ManifestHeader: View {
    @Binding var loadingMode: Bool
    let pendingMissionsCount: Int
    
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(loadingMode ? "LOADING SEQUENCE" : "UPCOMING")
                .font(.system(size: 10, weight: .heavy, design: .monospaced))
                .foregroundStyle(loadingMode ? LabTheme.fg : LabTheme.fgTertiary)
                .tracking(1.2)
            
            HStack(alignment: .top) {
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    Text(loadingMode ? "Loading Manifest" : "Route Manifest")
                        .font(.system(size: 28, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    
                    if pendingMissionsCount > 0 {
                        Text("\(pendingMissionsCount)")
                            .font(.system(size: 12, weight: .bold))
                            .foregroundStyle(LabTheme.buttonFg)
                            .frame(width: 24, height: 24)
                            .background(LabTheme.fg, in: Circle())
                    }
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 4) {
                    Text("mobile_driver.ui.loading_mode")
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(loadingMode ? LabTheme.fg : LabTheme.fgTertiary)
                        .tracking(0.8)
                    Toggle("", isOn: $loadingMode)
                        .labelsHidden()
                        .tint(LabTheme.fg)
                }
            }
        }
        .padding(.horizontal, LabTheme.s20)
        .padding(.top, 60)
        .padding(.bottom, LabTheme.s20)
    }
}
