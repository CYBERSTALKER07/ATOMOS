//
//  BatteryGuard.swift
//  driverappios
//

import UIKit

enum BatteryGuard {
    static let warnThreshold = 20
    static let blockThreshold = 10

    enum DepartGate {
        case ok
        case warnLow(level: Int)
        case blockCritical(level: Int)
    }

    static var levelPercent: Int {
        let device = UIDevice.current
        device.isBatteryMonitoringEnabled = true
        let level = device.batteryLevel
        if level < 0 { return 100 }
        return Int(level * 100)
    }

    static func departGate() -> DepartGate {
        let level = levelPercent
        if level < blockThreshold { return .blockCritical(level: level) }
        if level < warnThreshold { return .warnLow(level: level) }
        return .ok
    }
}
