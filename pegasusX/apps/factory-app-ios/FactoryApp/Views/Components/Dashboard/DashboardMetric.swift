import SwiftUI

struct DashboardMetric {
    let title: String
    let value: String
    let supporting: String
    let icon: String
    let tint: Color
    var chip: (text: String, tint: Color)? = nil
}
