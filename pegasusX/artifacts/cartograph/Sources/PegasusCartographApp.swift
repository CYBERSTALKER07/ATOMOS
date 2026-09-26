import AppKit
import SwiftUI
import UniformTypeIdentifiers

@main
struct PegasusCartographApp: App {
    init() {
        Snapshot.writeIfRequested()
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
        .windowStyle(.titleBar)
        .windowToolbarStyle(.unifiedCompact(showsTitle: true))
        .defaultSize(width: 1560, height: 960)
        .commands {
            CommandGroup(replacing: .newItem) {}
            CommandGroup(after: .saveItem) {
                Button("Export Map PNG…") {
                    Snapshot.exportInteractive()
                }
                .keyboardShortcut("e", modifiers: [.command, .shift])
            }
        }
    }
}

@MainActor
enum Snapshot {
    static func writeIfRequested() {
        let env = ProcessInfo.processInfo.environment
        guard let path = env["CARTOGRAPH_SNAPSHOT"], !path.isEmpty else { return }
        write(to: URL(fileURLWithPath: path))
        if env["CARTOGRAPH_SNAPSHOT_EXIT"] == "1" {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.4) {
                NSApp.terminate(nil)
            }
        }
    }

    static func exportInteractive() {
        let panel = NSSavePanel()
        panel.allowedContentTypes = [.png]
        panel.nameFieldStringValue = "pegasusx-cartograph-\(Atlas.commit).png"
        panel.begin { resp in
            guard resp == .OK, let url = panel.url else { return }
            write(to: url)
        }
    }

    static func write(to url: URL) {
        let view = ContentView()
            .frame(width: 1560, height: 960)
            .clipped()
        let renderer = ImageRenderer(content: view)
        renderer.scale = 2
        renderer.proposedSize = ProposedViewSize(width: 1560, height: 960)
        guard let image = renderer.nsImage else { return }
        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff),
              let png = rep.representation(using: .png, properties: [:]) else { return }
        try? png.write(to: url)
    }
}
