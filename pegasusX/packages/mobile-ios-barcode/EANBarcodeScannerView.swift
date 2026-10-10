//
//  EANBarcodeScannerView.swift
//  Shared EAN scanner for return-gate surfaces (payload + warehouse iOS).
//  VisionKit DataScanner (iOS 16+) with AVFoundation EAN fallback.
//

import AVFoundation
import SwiftUI
import VisionKit

private let eanScanDebounceSeconds: TimeInterval = 0.3

struct EANBarcodeScannerView: View {
    let onBarcode: (String) -> Void
    var enabled: Bool = true
    var previewHeight: CGFloat = 160
    var showTorchToggle: Bool = true

    @State private var torchOn = false

    private var canUseDataScanner: Bool {
        DataScannerViewController.isSupported && DataScannerViewController.isAvailable
    }

    var body: some View {
        VStack(spacing: 6) {
            Group {
                if !enabled {
                    Text("Scanner paused")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity)
                        .frame(height: previewHeight)
                } else if canUseDataScanner {
                    DataScannerRepresentable(onBarcode: onBarcode, torchOn: torchOn)
                        .frame(height: previewHeight)
                } else {
                    EANCameraPreview(onBarcode: onBarcode, torchOn: $torchOn)
                        .frame(height: previewHeight)
                }
            }
            .clipShape(.rect(cornerRadius: 8))

            if showTorchToggle && enabled {
                Button(torchOn ? "Torch on" : "Torch") {
                    torchOn.toggle()
                }
                .font(.caption)
                .buttonStyle(.bordered)
            }

            KeyboardWedgeBarcodeField(onBarcode: onBarcode, enabled: enabled)
        }
    }
}

/// Hidden field for hardware keyboard / Bluetooth wedge scanners.
struct KeyboardWedgeBarcodeField: View {
    let onBarcode: (String) -> Void
    var enabled: Bool = true

    @State private var buffer = ""
    @FocusState private var focused: Bool

    var body: some View {
        TextField("", text: $buffer)
            .focused($focused)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .submitLabel(.done)
            .onSubmit(emit)
            .onChange(of: buffer) { _, next in
                if next.contains("\n") || next.contains("\r") {
                    emit()
                }
            }
            .frame(width: 1, height: 1)
            .opacity(0.01)
            .accessibilityHidden(true)
            .disabled(!enabled)
            .onAppear {
                if enabled { focused = true }
            }
            .onChange(of: enabled) { _, on in
                if on { focused = true }
            }
    }

    private func emit() {
        let code = buffer
            .replacingOccurrences(of: "\r", with: "")
            .replacingOccurrences(of: "\n", with: "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        buffer = ""
        guard !code.isEmpty else { return }
        onBarcode(code)
    }
}

// MARK: - VisionKit DataScanner

private struct DataScannerRepresentable: UIViewControllerRepresentable {
    let onBarcode: (String) -> Void
    var torchOn: Bool

    func makeUIViewController(context: Context) -> DataScannerViewController {
        let controller = DataScannerViewController(
            recognizedDataTypes: [.barcode(symbologies: [.ean13, .ean8])],
            qualityLevel: .balanced,
            recognizesMultipleItems: false,
            isHighFrameRateTrackingEnabled: true,
            isHighlightingEnabled: true
        )
        controller.delegate = context.coordinator
        return controller
    }

    func updateUIViewController(_ uiViewController: DataScannerViewController, context: Context) {
        if !uiViewController.isScanning {
            try? uiViewController.startScanning()
        }
        // DataScanner torch API varies by OS; AVFoundation path handles torch when fallback used.
        _ = torchOn
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onBarcode: onBarcode)
    }

    final class Coordinator: NSObject, DataScannerViewControllerDelegate {
        let onBarcode: (String) -> Void
        private var lastCode = ""
        private var lastEmitAt: TimeInterval = 0

        init(onBarcode: @escaping (String) -> Void) {
            self.onBarcode = onBarcode
        }

        func dataScanner(_ dataScanner: DataScannerViewController, didTapOn item: RecognizedItem) {
            process(item)
        }

        func dataScanner(_ dataScanner: DataScannerViewController, didAdd addedItems: [RecognizedItem], allItems: [RecognizedItem]) {
            for item in addedItems {
                process(item)
            }
        }

        private func process(_ item: RecognizedItem) {
            guard case .barcode(let barcode) = item,
                  let payload = barcode.payloadStringValue else { return }
            let now = Date().timeIntervalSince1970
            guard payload != lastCode || (now - lastEmitAt) >= eanScanDebounceSeconds else { return }
            lastCode = payload
            lastEmitAt = now
            haptic()
            onBarcode(payload)
        }

        private func haptic() {
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        }
    }
}

// MARK: - AVFoundation fallback

private struct EANCameraPreview: UIViewRepresentable {
    let onBarcode: (String) -> Void
    @Binding var torchOn: Bool

    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.backgroundColor = .black

        let session = AVCaptureSession()
        context.coordinator.session = session

        guard let device = AVCaptureDevice.default(for: .video),
              let input = try? AVCaptureDeviceInput(device: device) else {
            return view
        }
        context.coordinator.device = device

        if session.canAddInput(input) {
            session.addInput(input)
        }

        let output = AVCaptureMetadataOutput()
        if session.canAddOutput(output) {
            session.addOutput(output)
            output.setMetadataObjectsDelegate(context.coordinator, queue: .main)
            output.metadataObjectTypes = [.ean13, .ean8]
        }

        let previewLayer = AVCaptureVideoPreviewLayer(session: session)
        previewLayer.videoGravity = .resizeAspectFill
        previewLayer.frame = view.bounds
        view.layer.addSublayer(previewLayer)
        context.coordinator.previewLayer = previewLayer

        DispatchQueue.global(qos: .userInitiated).async {
            session.startRunning()
        }

        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {
        context.coordinator.previewLayer?.frame = uiView.bounds
        context.coordinator.setTorch(torchOn)
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onBarcode: onBarcode)
    }

    final class Coordinator: NSObject, AVCaptureMetadataOutputObjectsDelegate {
        let onBarcode: (String) -> Void
        var session: AVCaptureSession?
        var previewLayer: AVCaptureVideoPreviewLayer?
        var device: AVCaptureDevice?
        private var lastCode = ""
        private var lastEmitAt: TimeInterval = 0

        init(onBarcode: @escaping (String) -> Void) {
            self.onBarcode = onBarcode
        }

        func setTorch(_ on: Bool) {
            guard let device, device.hasTorch else { return }
            do {
                try device.lockForConfiguration()
                device.torchMode = on ? .on : .off
                device.unlockForConfiguration()
            } catch {
                // ignore torch failures
            }
        }

        func metadataOutput(
            _ output: AVCaptureMetadataOutput,
            didOutput metadataObjects: [AVMetadataObject],
            from connection: AVCaptureConnection
        ) {
            guard let object = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
                  let value = object.stringValue else { return }
            let now = Date().timeIntervalSince1970
            guard value != lastCode || (now - lastEmitAt) >= eanScanDebounceSeconds else { return }
            lastCode = value
            lastEmitAt = now
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            onBarcode(value)
        }
    }
}
