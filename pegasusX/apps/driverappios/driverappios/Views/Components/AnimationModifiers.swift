//
//  AnimationModifiers.swift
//  driverappios
//

import SwiftUI

// MARK: - Pressable Button Style (scale on press)

struct PressableButtonStyle: ButtonStyle {
    var scale: Double = 0.96

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? scale : 1)
            .opacity(configuration.isPressed ? 0.85 : 1)
            .animation(Anim.micro, value: configuration.isPressed)
            .sensoryFeedback(.impact(weight: .light), trigger: configuration.isPressed)
    }
}

extension ButtonStyle where Self == PressableButtonStyle {
    static var pressable: PressableButtonStyle { PressableButtonStyle() }
    static func pressable(scale: Double) -> PressableButtonStyle {
        PressableButtonStyle(scale: scale)
    }
}

// MARK: - Slide Up Transition

extension AnyTransition {
    static var slideUp: AnyTransition {
        .asymmetric(
            insertion: .move(edge: .bottom).combined(with: .opacity),
            removal: .move(edge: .bottom).combined(with: .opacity)
        )
    }

    static var fadeScale: AnyTransition {
        .asymmetric(
            insertion: .scale(scale: 0.92).combined(with: .opacity),
            removal: .scale(scale: 0.95).combined(with: .opacity)
        )
    }
}

// MARK: - Shimmer Modifier

struct ShimmerModifier: ViewModifier {
    @State private var phase: Double = 0

    func body(content: Content) -> some View {
        content
            .overlay {
                LinearGradient(
                    colors: [.clear, .white.opacity(0.08), .clear],
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .offset(x: phase)
                .mask { content }
            }
            .onAppear {
                withAnimation(.linear(duration: 2).repeatForever(autoreverses: false)) {
                    phase = 300
                }
            }
    }
}

extension View {
    func shimmer() -> some View {
        modifier(ShimmerModifier())
    }
}

// MARK: - Staggered Appear Modifier

struct StaggeredAppear: ViewModifier {
    let index: Int
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var appeared = false

    func body(content: Content) -> some View {
        content
            .opacity(appeared ? 1 : 0)
            .offset(y: appeared ? 0 : (reduceMotion ? 0 : 20))
            .onAppear {
                if reduceMotion {
                    appeared = true
                } else {
                    withAnimation(Anim.stagger(index)) {
                        appeared = true
                    }
                }
            }
    }
}

extension View {
    func staggeredAppear(index: Int) -> some View {
        modifier(StaggeredAppear(index: index))
    }
}

// MARK: - Glow Pulse

struct GlowPulse: ViewModifier {
    let color: Color
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pulse = false

    func body(content: Content) -> some View {
        content
            .shadow(color: color.opacity(pulse ? 0.6 : 0.1), radius: pulse ? 12 : 4)
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(Anim.breathe) {
                    pulse = true
                }
            }
    }
}

extension View {
    func glowPulse(color: Color) -> some View {
        modifier(GlowPulse(color: color))
    }
}

// MARK: - Lab Card Style

struct LabCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .background {
                RoundedRectangle(cornerRadius: LabTheme.cardRadius)
                    .fill(LabTheme.card)
                    .stroke(LabTheme.separator, lineWidth: 0.5)
                    .shadow(color: .black.opacity(0.06), radius: 12, y: 6)
            }
    }
}

extension View {
    func labCard() -> some View {
        modifier(LabCardModifier())
    }
}

#Preview {
    VStack(spacing: 20) {
        Text("Pressable")
            .padding()
            .background(.black, in: .rect(cornerRadius: 12))
            .foregroundStyle(.white)

        Text("Shimmer")
            .font(.title.bold())
            .shimmer()

        ForEach(0..<3, id: \.self) { i in
            Text("Stagger \(i)")
                .padding()
                .labCard()
                .staggeredAppear(index: i)
        }
    }
    .padding()
}
