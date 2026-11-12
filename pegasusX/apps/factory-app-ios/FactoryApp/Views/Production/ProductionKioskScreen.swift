import SwiftUI

struct ProductionKioskScreen: View {
    let machineId: String = "MAC-123"
    @State private var isJammed: Bool = false
    @State private var currentStatus: String = "IDLE"

    var body: some View {
        ZStack {
            (isJammed ? Color.red.opacity(0.8) : Color(UIColor.systemBackground))
                .ignoresSafeArea()

            VStack(spacing: 40) {
                Text("Production Kiosk - \(machineId)")
                    .font(.title)
                    .foregroundStyle(isJammed ? .white : .primary)

                Text("STATUS: \(currentStatus)")
                    .font(.largeTitle.weight(.bold))
                    .foregroundStyle(isJammed ? .white : .primary)
                    .accessibilityLabel("Current machine status: \(currentStatus)")

                HStack(spacing: 40) {
                    Button {
                        PegasusHaptics.confirm()
                        currentStatus = "IN_PRODUCTION"
                        isJammed = false
                    } label: {
                        Text("START RUN")
                            .font(.title2.weight(.bold))
                            .foregroundStyle(.white)
                            .frame(minWidth: 250, minHeight: 150)
                            .background(Color.green.opacity(0.7))
                            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    }
                    .accessibilityLabel("Start production run")

                    Button {
                        PegasusHaptics.selection()
                        currentStatus = "PAUSED"
                        isJammed = false
                    } label: {
                        Text("PAUSE")
                            .font(.title2.weight(.bold))
                            .foregroundStyle(.white)
                            .frame(minWidth: 250, minHeight: 150)
                            .background(Color.yellow.opacity(0.7))
                            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    }
                    .accessibilityLabel("Pause production run")

                    Button {
                        PegasusHaptics.warning()
                        currentStatus = "JAMMED"
                        isJammed = true
                    } label: {
                        Text("FLAG ISSUE")
                            .font(.title2.weight(.bold))
                            .foregroundStyle(.white)
                            .frame(minWidth: 250, minHeight: 150)
                            .background(Color.red.opacity(0.9))
                            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    }
                    .accessibilityLabel("Flag machine jam or issue")
                }
            }
            .padding(32)
        }
    }
}
