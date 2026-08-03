import SwiftUI

struct ResponsiveGridView<Data: RandomAccessCollection, Content: View>: View where Data.Element: Identifiable {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let data: Data
    @ViewBuilder let content: (Data.Element) -> Content

    var body: some View {
        if horizontalSizeClass == .regular {
            ScrollView {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 340), spacing: 16)], spacing: 16) {
                    ForEach(data) { item in
                        content(item)
                            .padding()
                            .background(Color(UIColor.secondarySystemGroupedBackground))
                            .cornerRadius(10)
                    }
                }
                .padding()
            }
            .background(Color(UIColor.systemGroupedBackground))
        } else {
            ResponsiveGridContentWrapper {
                ForEach(data) { item in
                    content(item)
                }
            }
        }
    }
}
