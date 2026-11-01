//
//  SupplierSessionReconcile.swift
//  SupplierApp
//
//  Refetch server-authoritative supplier snapshots after transport reconnect.
//

import Foundation

enum SupplierSessionReconcile {
    static func run() async {
        async let previewTask = SupplierOperationsService.dispatchPreview()
        async let manifestsTask = SupplierOperationsService.manifests()
        _ = try? await previewTask
        _ = try? await manifestsTask
    }
}
