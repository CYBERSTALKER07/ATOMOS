//
//  WarehouseSessionReconcile.swift
//  WarehouseApp
//
//  Refetch server-authoritative warehouse snapshots after transport reconnect.
//

import Foundation

enum WarehouseSessionReconcile {
    static func run() async {
        _ = try? await WarehouseService.dispatchPreview()
        _ = try? await WarehouseService.dispatchLocks()
    }
}
