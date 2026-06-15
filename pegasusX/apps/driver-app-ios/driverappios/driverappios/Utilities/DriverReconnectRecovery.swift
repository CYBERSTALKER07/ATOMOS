//
//  DriverReconnectRecovery.swift
//  driverappios
//
//  Clears in-flight mutation spinners after transport reconnect.
//

import Foundation

enum DriverReconnectRecovery {
    static let hint = "Connection restored — verify status before retrying."

    static func recoverInFlight(wasInFlight: Bool) async {
        await DriverSessionReconcile.run()
    }
}
