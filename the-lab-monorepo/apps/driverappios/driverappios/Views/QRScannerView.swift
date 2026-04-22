//
//  QRScannerView.swift
//  driverappios
//

import AVFoundation
import SwiftUI

struct QRScannerView: View {
    let onValidated: (ValidateQRResponse) -> Void
    let onCancel: () -> Void

    @State private var vm = ScannerViewModel()
    @State private var cameraPermissionDenied = false

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            // MARK: - Camera Preview
            QRCameraPreview(onScan: { value in
                vm.handleScan(value, onValidated: onValidated)
            })
            .ignoresSafeArea()

            // MARK: - Overlay
            VStack {
                // Cancel button
                HStack {
                    Button {
                        onCancel()
                    } label: {
                        Text("Cancel")
                            .font(.body.weight(.medium))
                            .foregroundStyle(.white)
                            .padding(12)
                    }
                    Spacer()
                }
                .padding(.top, 8)

                Spacer()

                // Targeting reticle
                ZStack {
                    // Dimmed overlay with cutout
                    reticleOverlay
                }

                Spacer()

                // Processing indicator
                if vm.isProcessing {
                    HStack(spacing: 10) {
                        ProgressView()
                            .tint(.white)
                        Text("Processing...")
                            .font(.subheadline)
                            .foregroundStyle(.white)
                    }
                    .padding(.bottom, 40)
                }
            }
        }
        .task {
            let granted = await vm.checkCameraPermission()
            if !granted {
                cameraPermissionDenied = true
            }
        }
        .alert("Camera Access Required", isPresented: $cameraPermissionDenied) {
            Button("Close", role: .cancel) { onCancel() }
        } message: {
            Text("Please enable camera access in Settings to scan QR codes.")
        }
        .alert(vm.alertTitle, isPresented: $vm.showAlert) {
            if vm.scanSucceeded {
                Button("OK") { }
            } else {
                Button("Rescan", role: .cancel) { }
                Button("Close") { onCancel() }
            }
        } message: {
            Text(vm.alertMessage)
        }
    }

    // MARK: - Reticle

    private var reticleOverlay: some View {
        ZStack {
            // Corner brackets
            RoundedRectangle(cornerRadius: 16)
                .stroke(.white, lineWidth: 3)
                .frame(width: 240, height: 240)

            // Subtle inner area
            RoundedRectangle(cornerRadius: 16)
                .fill(.white.opacity(0.05))
                .frame(width: 240, height: 240)
        }
    }
}

// MARK: - AVFoundation Camera Preview

struct QRCameraPreview: UIViewRepresentable {
    let onScan: (String) -> Void

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
            output.metadataObjectTypes = [.qr]
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
        Coordinator(onScan: onScan)
    }

    class Coordinator: NSObject, AVCaptureMetadataOutputObjectsDelegate {
        let onScan: (String) -> Void
        var session: AVCaptureSession?
        var previewLayer: AVCaptureVideoPreviewLayer?

        init(onScan: @escaping (String) -> Void) {
            self.onScan = onScan
        }

        func metadataOutput(
            _ output: AVCaptureMetadataOutput,
            didOutput metadataObjects: [AVMetadataObject],
            from connection: AVCaptureConnection
        ) {
            guard let object = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
                  object.type == .qr,
                  let value = object.stringValue else { return }
            onScan(value)
        }
    }
}

#Preview {
    QRScannerView(onValidated: { _ in }, onCancel: {})
}
