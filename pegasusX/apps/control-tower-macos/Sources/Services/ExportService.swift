import AppKit
import Foundation

@MainActor
public enum ExportService {
    public static func exportDiagnosticBundle(bundle: DiagnosticBundle) {
        let panel = NSSavePanel()
        panel.title = "Export Diagnostic Bundle"
        panel.nameFieldStringValue = "pegasusx-diagnostic-\(bundle.bundleId).json"
        panel.canCreateDirectories = true

        panel.begin { response in
            guard response == .OK, let url = panel.url else { return }
            do {
                let encoder = JSONEncoder()
                encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
                let data = try encoder.encode(bundle)
                try data.write(to: url)
            } catch {
                let alert = NSAlert()
                alert.messageText = "Export Failed"
                alert.informativeText = error.localizedDescription
                alert.alertStyle = .warning
                alert.runModal()
            }
        }
    }

    public static func exportWaterfallCSV(slices: [TenderWaterfallSlice], role: RolePerspective) {
        let panel = NSSavePanel()
        panel.title = "Export Tender Waterfall CSV"
        panel.nameFieldStringValue = "tender-waterfall-\(role.rawValue.lowercased().replacingOccurrences(of: " ", with: "-")).csv"
        panel.canCreateDirectories = true

        panel.begin { response in
            guard response == .OK, let url = panel.url else { return }
            var csv = "Stage,Amount (Tiyin),Amount (UZS),Percentage,Type,Description\n"
            for slice in slices {
                let type = slice.isDeduction ? "Deduction" : "Gross/Payout"
                csv += "\"\(slice.label)\",\(slice.amountTiyin),\"\(slice.amountUzsFormatted)\",\(slice.percentage),\(type),\"\(slice.detail)\"\n"
            }
            do {
                try csv.write(to: url, atomically: true, encoding: .utf8)
            } catch {
                let alert = NSAlert()
                alert.messageText = "Export Failed"
                alert.informativeText = error.localizedDescription
                alert.alertStyle = .warning
                alert.runModal()
            }
        }
    }
}
