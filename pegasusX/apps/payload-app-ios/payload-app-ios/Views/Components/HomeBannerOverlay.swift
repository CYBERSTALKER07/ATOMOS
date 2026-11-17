import SwiftUI

struct HomeBannerOverlay: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        if viewModel.barcodeScanMessage != nil ||
           viewModel.missingItemsReportedMessage != nil ||
           viewModel.queuedNoticeMessage != nil ||
           viewModel.syncCompleteMessage != nil {
            VStack(spacing: 8) {
                if let msg = viewModel.barcodeScanMessage {
                    InfoBanner(text: msg, tint: .blue)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                }
                if let msg = viewModel.missingItemsReportedMessage {
                    InfoBanner(text: msg, tint: .orange)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                }
                if let msg = viewModel.queuedNoticeMessage {
                    InfoBanner(text: msg, tint: .orange)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                }
                if let msg = viewModel.syncCompleteMessage {
                    InfoBanner(text: msg, tint: .green)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                }
            }
            .animation(.easeInOut, value: viewModel.queuedNoticeMessage)
            .animation(.easeInOut, value: viewModel.syncCompleteMessage)
            .animation(.easeInOut, value: viewModel.barcodeScanMessage)
            .animation(.easeInOut, value: viewModel.missingItemsReportedMessage)
            .padding()
        }
    }
}

struct InfoBanner: View {
    let text: String
    let tint: Color

    var body: some View {
        Text(text)
            .font(.footnote.bold())
            .foregroundStyle(.white)
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(tint)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .shadow(radius: 4)
            .accessibilityElement(children: .combine)
            .accessibilityLabel(text)
    }
}
