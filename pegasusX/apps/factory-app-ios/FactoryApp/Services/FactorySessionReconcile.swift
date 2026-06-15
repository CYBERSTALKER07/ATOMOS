//
//  FactorySessionReconcile.swift
//  FactoryApp
//
//  Refetch server-authoritative factory snapshots after transport reconnect.
//

import Foundation

enum FactorySessionReconcile {
    static func run() async {
        _ = try? await FactoryService.manifests()
    }
}
