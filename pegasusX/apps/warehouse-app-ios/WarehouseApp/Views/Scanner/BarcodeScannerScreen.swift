import SwiftUI
import AVFoundation
import AudioToolbox

struct BarcodeScannerScreen: View {
    @State private var scanState: ScanState = .idle
    @State private var scannedBins: [String] = []
    
    enum ScanState {
        case idle
        case success
        case error
    }
    
    var body: some View {
        VStack {
            Text("SCAN BIN BARCODE")
                .font(.largeTitle)
                .bold()
                .padding()
            
            Spacer()
            
            ZStack {
                Rectangle()
                    .fill(scanState == .success ? Color.green.opacity(0.3) : (scanState == .error ? Color.red.opacity(0.3) : Color.gray.opacity(0.1)))
                    .frame(height: 300)
                    .cornerRadius(12)
                
                Text(scanState == .success ? "SUCCESS" : (scanState == .error ? "ERROR" : "READY"))
                    .font(.title)
                    .foregroundColor(scanState == .success ? .green : (scanState == .error ? .red : .gray))
            }
            .padding()
            
            Spacer()
            
            Button("SIMULATE SUCCESS SCAN") {
                triggerSuccessFeedback()
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(Color.green)
            .foregroundColor(.white)
            .cornerRadius(8)
            .padding(.horizontal)
            
            Button("SIMULATE ERROR SCAN") {
                triggerErrorFeedback()
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(Color.red)
            .foregroundColor(.white)
            .cornerRadius(8)
            .padding(.horizontal)
        }
    }
    
    private func triggerSuccessFeedback() {
        scanState = .success
        // Haptic
        let generator = UINotificationFeedbackGenerator()
        generator.notificationOccurred(.success)
        // Acoustic (high pitch beep)
        AudioServicesPlaySystemSound(1052)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
            scanState = .idle
        }
    }
    
    private func triggerErrorFeedback() {
        scanState = .error
        // Haptic
        let generator = UINotificationFeedbackGenerator()
        generator.notificationOccurred(.error)
        // Acoustic (low pitch buzz)
        AudioServicesPlaySystemSound(1053)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
            scanState = .idle
        }
    }
}
