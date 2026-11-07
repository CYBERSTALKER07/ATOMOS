import Foundation

@Observable
final class InsightsViewModel {
    var analytics: RetailerAnalytics?
    var detailed: RetailerDetailedAnalytics?
    var predictions: [RetailerAIPrediction] = []
    var isLoading = false
    var selectedRange: DateRange = .month
    var correctingId: String?

    private let api = APIClient.shared

    var delta: Int {
        guard let a = analytics else { return 0 }
        guard a.totalLastMonth > 0 else { return 0 }
        return Int(Double(a.totalThisMonth - a.totalLastMonth) / Double(a.totalLastMonth) * 100)
    }

    func loadAnalytics() async {
        isLoading = true
        defer { isLoading = false }
        do {
            analytics = try await api.get(path: "/v1/retailer/analytics/expenses")
        } catch {
            analytics = RetailerAnalytics.sample
        }
        predictions = RetailerAIPrediction.samples
    }

    private static let fullDateFormatter: ISO8601DateFormatter = {
        let fmt = ISO8601DateFormatter()
        fmt.formatOptions = [.withFullDate]
        return fmt
    }()

    func loadDetailedAnalytics() async {
        let to = Date()
        let from = Calendar.current.date(byAdding: .day, value: -selectedRange.days, to: to) ?? to
        let fromStr = Self.fullDateFormatter.string(from: from)
        let toStr = Self.fullDateFormatter.string(from: to)
        do {
            detailed = try await api.get(path: "/v1/retailer/analytics/detailed?from=\(fromStr)&to=\(toStr)")
        } catch {
            detailed = RetailerDetailedAnalytics.sample
        }
    }
}

// MARK: - Mock Fallback Samples

extension RetailerAnalytics {
    static let sample = RetailerAnalytics(
        monthlyExpenses: [
            MonthlyExpense(month: "2026-01", total: 42_000_000),
            MonthlyExpense(month: "2026-02", total: 48_500_000),
            MonthlyExpense(month: "2026-03", total: 54_200_000)
        ],
        topSuppliers: [
            TopSupplierExpense(supplierID: "sup-001", supplierName: "Fresh Farms Co.", total: 24_500_000, orderCount: 14),
            TopSupplierExpense(supplierID: "sup-002", supplierName: "Bakery Express", total: 18_200_000, orderCount: 9),
            TopSupplierExpense(supplierID: "sup-003", supplierName: "Mountain Spring Water", total: 11_500_000, orderCount: 6)
        ],
        topProducts: [
            TopProductExpense(productID: "prod-001", productName: "Organic Whole Milk", total: 12_400_000, quantity: 420),
            TopProductExpense(productID: "prod-002", productName: "Sourdough Bread", total: 8_900_000, quantity: 180),
            TopProductExpense(productID: "prod-003", productName: "Free-Range Eggs", total: 7_600_000, quantity: 240)
        ],
        totalThisMonth: 54_200_000,
        totalLastMonth: 48_500_000
    )
}

extension RetailerDetailedAnalytics {
    static let sample = RetailerDetailedAnalytics(
        dailySpending: [
            RetailerDayExpense(date: "2026-03-10", total: 6_200_000, count: 4),
            RetailerDayExpense(date: "2026-03-11", total: 8_100_000, count: 5),
            RetailerDayExpense(date: "2026-03-12", total: 5_400_000, count: 3),
            RetailerDayExpense(date: "2026-03-13", total: 9_800_000, count: 6),
            RetailerDayExpense(date: "2026-03-14", total: 11_200_000, count: 7),
            RetailerDayExpense(date: "2026-03-15", total: 7_500_000, count: 4),
            RetailerDayExpense(date: "2026-03-16", total: 6_000_000, count: 5)
        ],
        ordersByState: [
            OrderStateCount(state: "DELIVERED", count: 28),
            OrderStateCount(state: "IN_TRANSIT", count: 4),
            OrderStateCount(state: "CONFIRMED", count: 2)
        ],
        categoryBreakdown: [
            CategorySpend(category: "Dairy & Eggs", total: 24_500_000, count: 18),
            CategorySpend(category: "Bakery", total: 16_800_000, count: 12),
            CategorySpend(category: "Beverages", total: 12_900_000, count: 8)
        ],
        weekdayPattern: [
            DayOfWeekPattern(weekday: "Mon", avg: 7_400_000, count: 4),
            DayOfWeekPattern(weekday: "Tue", avg: 8_100_000, count: 4),
            DayOfWeekPattern(weekday: "Wed", avg: 6_900_000, count: 4),
            DayOfWeekPattern(weekday: "Thu", avg: 9_200_000, count: 4),
            DayOfWeekPattern(weekday: "Fri", avg: 11_500_000, count: 4),
            DayOfWeekPattern(weekday: "Sat", avg: 8_300_000, count: 4),
            DayOfWeekPattern(weekday: "Sun", avg: 4_200_000, count: 4)
        ],
        totalSpent: 54_200_000,
        totalOrders: 34,
        avgOrderValue: 1_594_117
    )
}
