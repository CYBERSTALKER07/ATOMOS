import SwiftUI

enum AnimationConstants {
    /// Primary spring — cards, modals, scale effects
    static let fluid = Animation.spring(response: 0.3, dampingFraction: 0.8)

    /// Quick spring — buttons, small interactions
    static let express = Animation.spring(response: 0.15, dampingFraction: 0.9)

    /// Bouncy — playful interactions
    static let bouncy = Animation.spring(response: 0.35, dampingFraction: 0.6)

    /// Tab switch
    static let tabSwitch = Animation.easeOut(duration: 0.08)

    /// Numeric text transitions
    static let snappy = Animation.snappy

    /// Sheet presentation
    static let sheet = Animation.spring(response: 0.35, dampingFraction: 0.86)

    /// Stagger base delay
    static let staggerDelay: Double = 0.05

    /// Hero entrance
    static let hero = Animation.spring(response: 0.6, dampingFraction: 0.75)

    /// Micro interaction
    static let micro = Animation.spring(response: 0.1, dampingFraction: 0.95)

    /// Smooth ease
    static let smooth = Animation.easeInOut(duration: 0.25)
}
