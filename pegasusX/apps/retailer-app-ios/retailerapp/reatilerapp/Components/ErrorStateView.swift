import SwiftUI

struct ErrorStateView: View {
    var title: String = "Failed to Load"
    var message: String = "Could not load data. Check your connection and try again."
    var icon: String = "wifi.exclamationmark"
    var retryAction: (() async -> Void)?

    @State private var isRetrying = false

    var body: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            
            ZStack {
                Circle()
                    .fill(AppTheme.destructiveSoft.opacity(0.3))
                    .frame(width: 80, height: 80)
                Image(systemName: icon)
                    .font(.system(size: 32))
                    .foregroundStyle(AppTheme.destructive.opacity(0.8))
            }
            
            Text(title)
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
                .multilineTextAlignment(.center)
            
            Text(message)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, AppTheme.spacingXL)

            if let retryAction = retryAction {
                if isRetrying {
                    ProgressView()
                        .padding(.top, AppTheme.spacingMD)
                } else {
                    LabButton("Retry", variant: .outline) {
                        isRetrying = true
                        Task {
                            await retryAction()
                            isRetrying = false
                        }
                    }
                    .padding(.top, AppTheme.spacingMD)
                }
            }
            Spacer()
        }
        .padding(AppTheme.spacingLG)
    }
}

#Preview {
    ErrorStateView(retryAction: {})
}
