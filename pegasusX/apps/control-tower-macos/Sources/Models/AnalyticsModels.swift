import Foundation

public struct RolePerformanceScorecard: Identifiable, Codable, Sendable {
    public var id: String { role }
    public let role: String
    public let primaryMetricName: String
    public let primaryMetricValue: String
    public let primaryMetricSubtext: String
    public let secondaryMetricName: String
    public let secondaryMetricValue: String
    public let complianceRatePercent: Double
    public let activeAlertCount: Int

    public init(
        role: String,
        primaryMetricName: String,
        primaryMetricValue: String,
        primaryMetricSubtext: String,
        secondaryMetricName: String,
        secondaryMetricValue: String,
        complianceRatePercent: Double,
        activeAlertCount: Int
    ) {
        self.role = role
        self.primaryMetricName = primaryMetricName
        self.primaryMetricValue = primaryMetricValue
        self.primaryMetricSubtext = primaryMetricSubtext
        self.secondaryMetricName = secondaryMetricName
        self.secondaryMetricValue = secondaryMetricValue
        self.complianceRatePercent = complianceRatePercent
        self.activeAlertCount = activeAlertCount
    }
}
