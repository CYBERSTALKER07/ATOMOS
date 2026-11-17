import SwiftUI

struct ResponsiveGridContentWrapper<Content: View>: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @ViewBuilder let content: () -> Content

    var body: some View {
        GeometryReader { proxy in
            let width = proxy.size.width
            ScrollView {
                if width < 520 {
                    // Compact iPhone / narrow Split-View column: tactile single column stack
                    LazyVStack(spacing: 12) {
                        content()
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 12)
                } else if width < 860 {
                    // Medium iPad split screen / portrait
                    LazyVGrid(
                        columns: [
                            GridItem(.flexible(minimum: 240), spacing: 14),
                            GridItem(.flexible(minimum: 240), spacing: 14)
                        ],
                        spacing: 14
                    ) {
                        content()
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 14)
                } else {
                    // Expanded iPad Control Tower / 3-column stage
                    LazyVGrid(
                        columns: [GridItem(.adaptive(minimum: 300, maximum: 420), spacing: 16)],
                        spacing: 16
                    ) {
                        content()
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 16)
                }
            }
            .background(TacticalTheme.canvas)
        }
    }
}
