//
//  EANBarcodeScannerView.swift
//  Shared EAN scanner for return-gate surfaces (payload + warehouse iOS).
//  VisionKit DataScanner (iOS 16+) with AVFoundation EAN fallback.
//

import AVFoundation
import SwiftUI
import VisionKit

struct EANBarcodeScannerView: View {
    let onBarcode: (String) -> Void
    var enabled: Bool = true
    var previewHeight: CGFloat = 160

    private var canUseDataScanner: Bool {
        DataScannerViewController.isSupported && DataScannerViewController.isAvailable
    }

    var body: some View {
        Group {
            if !enabled {
                Text("Scanner paused")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .frame(height: previewHeight)
            } else if canUseDataScanner {
                DataScannerRepresentable(onBarcode: onBarcode)
                    .frame(height: previewHeight)
            } else {
                EANCameraPreview(onBarcode: onBarcode)
                    .frame(height: previewHeight)
            }
        }
        .clipShape(.rect(cornerRadius: 8))
    }
}

// MARK: - VisionKit DataScanner

private struct DataScannerRepresentable: UIViewControllerRepresentable {
    let onBarcode: (String) -> Void

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
            guard payload != lastCode || (now - lastEmitAt) >= 1.5 else { return }
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

    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.backgroundColor = .black

        let session = AVCaptureSession()
        context.coordinator.session = session

        guard let device = AVCaptureDevice.default(for: .video),
              let input = try? AVCaptureDeviceInput(device: device) else {
            return view
        }

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
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onBarcode: onBarcode)
    }

    final class Coordinator: NSObject, AVCaptureMetadataOutputObjectsDelegate {
        let onBarcode: (String) -> Void
        var session: AVCaptureSession?
        var previewLayer: AVCaptureVideoPreviewLayer?
        private var lastCode = ""
        private var lastEmitAt: TimeInterval = 0

        init(onBarcode: @escaping (String) -> Void) {
            self.onBarcode = onBarcode
        }

        func metadataOutput(
            _ output: AVCaptureMetadataOutput,
            didOutput metadataObjects: [AVMetadataObject],
            from connection: AVCaptureConnection
        ) {
            guard let object = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
                  let value = object.stringValue else { return }
            let now = Date().timeIntervalSince1970
            guard value != lastCode || (now - lastEmitAt) >= 1.5 else { return }
            lastCode = value
            lastEmitAt = now
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            onBarcode(value)
        }
    }
}
