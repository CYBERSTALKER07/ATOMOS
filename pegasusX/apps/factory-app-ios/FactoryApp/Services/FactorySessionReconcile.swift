//
//  FactorySessionReconcile.swift
//  FactoryApp
//
//  Refetch server-authoritative factory snapshots after transport reconnect.
//

import Foundation

enum FactorySessionReconcile {
    static func run() async {
        async let manifests = try? FactoryService.manifests()
        async let dashboard = try? FactoryService.dashboard()
        async let analytics = try? FactoryService.analyticsOverview()
        async let insights = try? FactoryService.insights()
        _ = await (manifests, dashboard, analytics, insights)
    }
}
