import SwiftUI

private let resolvableKinds: Set<String> = ["CASH_DISCREPANCY", "CREDIT_NOTE_DRAFT", "CREDIT_FREEZE"]

struct ExceptionsList: View {
    let rows: [SupplierExceptionRow]
    var busyKey: String? = nil
    var onResolve: (SupplierExceptionRow) -> Void = { _ in }

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(rows) { row in
                VStack(alignment: .leading, spacing: 6) {
                    Text(row.orderId).font(.headline)
                    Text("\(row.kind) · \(row.status)").font(.subheadline)
                    if let note = row.note, !note.isEmpty { Text(note).font(.caption) }
                    if let manifestId = row.manifestId { Text("Manifest \(manifestId)").font(.caption) }
                    if resolvableKinds.contains(row.kind.uppercased()) {
                        let key = "\(row.kind):\(row.orderId)"
                        Button("Resolve") {
                            onResolve(row)
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(busyKey == key)
                    }
                }
            }
        }
    }
}
