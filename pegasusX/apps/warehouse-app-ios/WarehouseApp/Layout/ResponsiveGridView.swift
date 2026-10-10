import SwiftUI

struct ResponsiveGridView<Data: RandomAccessCollection, Content: View>: View where Data.Element: Identifiable {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let data: Data
    @ViewBuilder let content: (Data.Element) -> Content

    var body: some View {
        GeometryReader { proxy in
            let width = proxy.size.width
            ScrollView {
                if width < 520 {
                    LazyVStack(spacing: 12) {
                        ForEach(data) { item in
                            content(item)
                        }
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 12)
                } else if width < 860 {
                    LazyVGrid(
                        columns: [
                            GridItem(.flexible(minimum: 240), spacing: 14),
                            GridItem(.flexible(minimum: 240), spacing: 14)
                        ],
                        spacing: 14
                    ) {
                        ForEach(data) { item in
                            content(item)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 14)
                } else {
                    LazyVGrid(
                        columns: [GridItem(.adaptive(minimum: 300, maximum: 420), spacing: 16)],
                        spacing: 16
                    ) {
                        ForEach(data) { item in
                            content(item)
                        }
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 16)
                }
            }
            .background(TacticalTheme.canvas)
        }
    }
}
