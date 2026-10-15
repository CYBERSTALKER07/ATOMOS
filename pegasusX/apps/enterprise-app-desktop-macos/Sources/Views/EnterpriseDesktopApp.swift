import SwiftUI

@main
public struct EnterpriseDesktopApp: App {
    public init() {}

    public var body: some Scene {
        WindowGroup("Pegasus Enterprise Console") {
            ContentView()
                .preferredColorScheme(.dark)
                .frame(minWidth: 900, minHeight: 600)
                .background(SwissCADTheme.pitchBlack)
        }
        .windowStyle(.titleBar)
        .windowToolbarStyle(.unified)
    }
}
