import SwiftUI
import UIKit

/// Ink pad for credit-leave PoD signatures. Calls onSave with JPEG data.
struct SignaturePadView: View {
    var onSave: (Data) -> Void

    @State private var strokes: [[CGPoint]] = []
    @State private var current: [CGPoint] = []

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Signature")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Canvas { context, _ in
                for stroke in strokes {
                    guard stroke.count >= 2 else { continue }
                    var path = Path()
                    path.addLines(stroke)
                    context.stroke(path, with: .color(.black), lineWidth: 3)
                }
                if current.count >= 2 {
                    var path = Path()
                    path.addLines(current)
                    context.stroke(path, with: .color(.black), lineWidth: 3)
                }
            }
            .frame(height: 140)
            .background(Color.white)
            .clipShape(.rect(cornerRadius: 8))
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(Color.secondary.opacity(0.4), lineWidth: 1)
            )
            .gesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { value in
                        current.append(value.location)
                    }
                    .onEnded { _ in
                        if current.count >= 2 {
                            strokes.append(current)
                        }
                        current = []
                    }
            )
            HStack {
                Button("Clear") {
                    strokes = []
                    current = []
                }
                .buttonStyle(.bordered)
                Spacer()
                Button("Save signature") {
                    if let data = renderJPEG() {
                        onSave(data)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(strokes.isEmpty)
            }
        }
    }

    private func renderJPEG() -> Data? {
        let size = CGSize(width: 640, height: 280)
        let renderer = UIGraphicsImageRenderer(size: size)
        let image = renderer.image { ctx in
            UIColor.white.setFill()
            ctx.fill(CGRect(origin: .zero, size: size))
            UIColor.black.setStroke()
            let scaleX = size.width / 320
            let scaleY = size.height / 140
            for stroke in strokes {
                guard stroke.count >= 2 else { continue }
                let path = UIBezierPath()
                path.lineWidth = 3
                path.lineCapStyle = .round
                path.lineJoinStyle = .round
                path.move(to: CGPoint(x: stroke[0].x * scaleX, y: stroke[0].y * scaleY))
                for p in stroke.dropFirst() {
                    path.addLine(to: CGPoint(x: p.x * scaleX, y: p.y * scaleY))
                }
                path.stroke()
            }
        }
        return image.jpegData(compressionQuality: 0.85)
    }
}
