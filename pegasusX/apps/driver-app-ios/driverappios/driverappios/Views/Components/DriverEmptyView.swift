import SwiftUI

struct DriverEmptyView: View {
    let icon: String
    let title: String
    let message: String
    var actionTitle: String? = nil
    var action: (() -> Void)? = nil

    var body: some View {
        DriverStateCard(
            icon: icon,
            title: title,
            message: message,
            actionTitle: actionTitle,
            action: action
        )
        .frame(maxWidth: .infinity, minHeight: 180)
        .padding(LabTheme.s24)
    }
}
