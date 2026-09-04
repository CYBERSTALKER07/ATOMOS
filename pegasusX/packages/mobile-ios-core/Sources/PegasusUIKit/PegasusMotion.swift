import SwiftUI

/// Shared motion aliases aligned with MDC 1.14 / Android PegasusMotionTokens.
public enum PegasusAnim {
  public static let snappy = Animation.snappy(duration: 0.3)
  public static let smooth = Animation.smooth(duration: 0.35)
  public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
  public static let quick = Animation.easeOut(duration: 0.15)

  public static func staggerDelay(index: Int) -> Double {
    min(Double(index) * 0.04, 0.4)
  }
}

public enum PegasusMotionDuration {
  public static let short4: Double = 0.2
  public static let medium2: Double = 0.3
  public static let medium4: Double = 0.4
}
