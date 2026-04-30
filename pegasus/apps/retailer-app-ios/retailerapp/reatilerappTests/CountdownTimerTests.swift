import Testing
import Foundation
@testable import reatilerapp

struct CountdownTimerTests {

    // MARK: - Initial state

    @Test func initialState_defaultString() {
        let timer = CountdownTimer()
        #expect(timer.remainingString == "--:--:--")
        #expect(timer.isExpired == false)
    }

    // MARK: - start(from isoString)

    @Test func startFromISO_setsTarget() {
        let timer = CountdownTimer()
        timer.start(from: "2030-01-01T00:00:00Z")
        // After starting, calling update should produce a non-default string
        timer.update(now: ISO8601DateFormatter().date(from: "2029-12-31T23:00:00Z")!)
        #expect(timer.remainingString == "01:00:00")
    }

    @Test func startFromISO_invalidString_noTarget() {
        let timer = CountdownTimer()
        timer.start(from: "not-a-date")
        timer.update(now: Date())
        #expect(timer.remainingString == "--:--:--")
    }

    // MARK: - start(from date)

    @Test func startFromDate_setsTarget() {
        let timer = CountdownTimer()
        let target = Date().addingTimeInterval(3600) // 1 hour from now
        timer.start(from: target)
        timer.update(now: Date())
        #expect(timer.remainingString != "--:--:--")
        #expect(timer.isExpired == false)
    }

    // MARK: - update(now:)

    @Test func update_exactlyOneHour() {
        let timer = CountdownTimer()
        let now = Date()
        let target = now.addingTimeInterval(3600)
        timer.start(from: target)
        timer.update(now: now)
        #expect(timer.remainingString == "01:00:00")
    }

    @Test func update_lessThanMinute() {
        let timer = CountdownTimer()
        let now = Date()
        let target = now.addingTimeInterval(45)
        timer.start(from: target)
        timer.update(now: now)
        #expect(timer.remainingString == "00:00:45")
    }

    @Test func update_expired() {
        let timer = CountdownTimer()
        let now = Date()
        let target = now.addingTimeInterval(-10) // 10 seconds ago
        timer.start(from: target)
        timer.update(now: now)
        #expect(timer.remainingString == "00:00:00")
        #expect(timer.isExpired == true)
    }

    @Test func update_noTarget_defaultString() {
        let timer = CountdownTimer()
        timer.update(now: Date())
        #expect(timer.remainingString == "--:--:--")
    }

    @Test func update_multipleHoursMinutesSeconds() {
        let timer = CountdownTimer()
        let now = Date()
        // 2 hours, 30 minutes, 15 seconds = 9015 seconds
        let target = now.addingTimeInterval(9015)
        timer.start(from: target)
        timer.update(now: now)
        #expect(timer.remainingString == "02:30:15")
    }

    @Test func update_transitionsToExpired() {
        let timer = CountdownTimer()
        let now = Date()
        let target = now.addingTimeInterval(5)
        timer.start(from: target)

        // Not expired yet
        timer.update(now: now)
        #expect(timer.isExpired == false)

        // Now expired
        timer.update(now: now.addingTimeInterval(10))
        #expect(timer.isExpired == true)
    }

    @Test func update_expiredThenNewTarget_resetsExpired() {
        let timer = CountdownTimer()
        let now = Date()

        // First: expired
        timer.start(from: now.addingTimeInterval(-10))
        timer.update(now: now)
        #expect(timer.isExpired == true)

        // New target in the future
        timer.start(from: now.addingTimeInterval(60))
        timer.update(now: now)
        #expect(timer.isExpired == false)
        #expect(timer.remainingString == "00:01:00")
    }
}
