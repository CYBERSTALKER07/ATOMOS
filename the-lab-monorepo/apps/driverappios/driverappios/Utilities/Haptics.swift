//
//  Haptics.swift
//  driverappios
//

import UIKit

/// Centralized haptic feedback for non-view contexts (ViewModels, services).
/// In SwiftUI views, prefer `.sensoryFeedback()` modifier instead.
enum Haptics {
    static func heavy()            { UIImpactFeedbackGenerator(style: .heavy).impactOccurred() }
    static func medium()           { UIImpactFeedbackGenerator(style: .medium).impactOccurred() }
    static func light()            { UIImpactFeedbackGenerator(style: .light).impactOccurred() }
    static func soft()             { UIImpactFeedbackGenerator(style: .soft).impactOccurred() }
    static func success()          { UINotificationFeedbackGenerator().notificationOccurred(.success) }
    static func error()            { UINotificationFeedbackGenerator().notificationOccurred(.error) }
    static func warning()          { UINotificationFeedbackGenerator().notificationOccurred(.warning) }
    static func selectionChanged() { UISelectionFeedbackGenerator().selectionChanged() }
}
