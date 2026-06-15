//
//  SupplierSessionReconcile.swift
//  SupplierApp
//
//  Refetch server-authoritative supplier snapshots after transport reconnect.
//

import Foundation

enum SupplierSessionReconcile {
    static func run() async {
        _ = try? await SupplierOperationsService.dispatchPreview()
        _ = try? await SupplierOperationsService.manifests()
    }
}
