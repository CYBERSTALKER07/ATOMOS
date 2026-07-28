import SwiftUI

struct ExceptionsList: View {
    let rows: [SupplierExceptionRow]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(rows) { row in
                VStack(alignment: .leading, spacing: 4) {
                    Text(row.orderId).font(.headline)
                    Text("\(row.kind) · \(row.status)").font(.subheadline)
                    if let note = row.note, !note.isEmpty { Text(note).font(.caption) }
                    if let manifestId = row.manifestId { Text("Manifest \(manifestId)").font(.caption) }
                }
            }
        }
    }
}
