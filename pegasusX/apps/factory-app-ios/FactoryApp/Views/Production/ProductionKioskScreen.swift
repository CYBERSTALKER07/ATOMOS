import SwiftUI

struct ProductionKioskScreen: View {
    let machineId: String = "MAC-123"
    @State private var isJammed: Bool = false
    @State private var currentStatus: String = "IDLE"
    
    var body: some View {
        ZStack {
            (isJammed ? Color.red.opacity(0.8) : Color(UIColor.systemBackground))
                .edgesIgnoringSafeArea(.all)
            
            VStack(spacing: 40) {
                Text("Production Kiosk - \(machineId)")
                    .font(.title)
                    .foregroundColor(isJammed ? .white : .primary)
                
                Text("STATUS: \(currentStatus)")
                    .font(.system(size: 48, weight: .bold))
                    .foregroundColor(isJammed ? .white : .primary)
                
                HStack(spacing: 40) {
                    Button(action: {
                        currentStatus = "IN_PRODUCTION"
                        isJammed = false
                    }) {
                        Text("START RUN")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundColor(.white)
                            .frame(width: 250, height: 150)
                            .background(Color.green.opacity(0.7))
                            .cornerRadius(16)
                    }
                    
                    Button(action: {
                        currentStatus = "PAUSED"
                        isJammed = false
                    }) {
                        Text("PAUSE")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundColor(.white)
                            .frame(width: 250, height: 150)
                            .background(Color.yellow.opacity(0.7))
                            .cornerRadius(16)
                    }
                    
                    Button(action: {
                        currentStatus = "JAMMED"
                        isJammed = true
                    }) {
                        Text("FLAG ISSUE")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundColor(.white)
                            .frame(width: 250, height: 150)
                            .background(Color.red.opacity(0.9))
                            .cornerRadius(16)
                    }
                }
            }
            .padding(32)
        }
    }
}
