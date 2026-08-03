import SwiftUI

struct SheetHandle: View {
    var body: some View {
        Capsule()
            .fill(LabTheme.fgTertiary.opacity(0.4))
            .frame(width: 32, height: 4)
            .frame(maxWidth: .infinity)
            .padding(.top, 10).padding(.bottom, 12)
    }
}
