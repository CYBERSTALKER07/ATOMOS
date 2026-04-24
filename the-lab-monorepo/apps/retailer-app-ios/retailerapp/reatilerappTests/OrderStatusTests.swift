import Testing
@testable import reatilerapp

struct OrderStatusTests {

    // MARK: - displayName

    @Test func displayName_pending() {
        #expect(OrderStatus.pending.displayName == "Order Placed")
    }

    @Test func displayName_loaded() {
        #expect(OrderStatus.loaded.displayName == "Approved")
    }

    @Test func displayName_dispatched() {
        #expect(OrderStatus.dispatched.displayName == "Dispatched")
    }

    @Test func displayName_inTransit() {
        #expect(OrderStatus.inTransit.displayName == "Active")
    }

    @Test func displayName_arrived() {
        #expect(OrderStatus.arrived.displayName == "Driver Arrived")
    }

    @Test func displayName_awaitingGlobalPaynt() {
        #expect(OrderStatus.awaitingGlobalPaynt.displayName == "Awaiting GlobalPaynt")
    }

    @Test func displayName_pendingCash() {
        #expect(OrderStatus.pendingCashCollection.displayName == "Cash Collection")
    }

    @Test func displayName_completed() {
        #expect(OrderStatus.completed.displayName == "Delivered")
    }

    @Test func displayName_cancelled() {
        #expect(OrderStatus.cancelled.displayName == "Cancelled")
    }

    // MARK: - color

    @Test func color_pending_isOrange() {
        #expect(OrderStatus.pending.color == "systemOrange")
    }

    @Test func color_completed_isGreen() {
        #expect(OrderStatus.completed.color == "systemGreen")
    }

    @Test func color_cancelled_isRed() {
        #expect(OrderStatus.cancelled.color == "systemRed")
    }

    // MARK: - isActive

    @Test func isActive_activeStates() {
        let active: [OrderStatus] = [.loaded, .dispatched, .inTransit, .arrived, .awaitingGlobalPaynt, .pendingCashCollection]
        for status in active {
            #expect(status.isActive == true, "\(status) should be active")
        }
    }

    @Test func isActive_nonActiveStates() {
        let inactive: [OrderStatus] = [.pending, .completed, .cancelled]
        for status in inactive {
            #expect(status.isActive == false, "\(status) should not be active")
        }
    }

    // MARK: - hasDeliveryToken

    @Test func hasDeliveryToken_dispatched() {
        #expect(OrderStatus.dispatched.hasDeliveryToken == true)
    }

    @Test func hasDeliveryToken_inTransit() {
        #expect(OrderStatus.inTransit.hasDeliveryToken == true)
    }

    @Test func hasDeliveryToken_arrived() {
        #expect(OrderStatus.arrived.hasDeliveryToken == true)
    }

    @Test func hasDeliveryToken_false_pending() {
        #expect(OrderStatus.pending.hasDeliveryToken == false)
    }

    @Test func hasDeliveryToken_false_completed() {
        #expect(OrderStatus.completed.hasDeliveryToken == false)
    }

    // MARK: - timelineStepIndex

    @Test func timelineStepIndex_ordered() {
        #expect(OrderStatus.pending.timelineStepIndex == 0)
        #expect(OrderStatus.loaded.timelineStepIndex == 1)
        #expect(OrderStatus.dispatched.timelineStepIndex == 2)
        #expect(OrderStatus.inTransit.timelineStepIndex == 3)
        #expect(OrderStatus.arrived.timelineStepIndex == 4)
        #expect(OrderStatus.completed.timelineStepIndex == 5)
        #expect(OrderStatus.cancelled.timelineStepIndex == -1)
    }

    // MARK: - timelineSteps

    @Test func timelineSteps_has6Entries() {
        #expect(OrderStatus.timelineSteps.count == 6)
    }

    @Test func timelineSteps_firstIsPlaced() {
        #expect(OrderStatus.timelineSteps.first == "Placed")
    }

    @Test func timelineSteps_lastIsDelivered() {
        #expect(OrderStatus.timelineSteps.last == "Delivered")
    }

    // MARK: - Order computed props

    @Test func order_isAiGenerated_true() {
        let order = Order(
            id: "o1", retailerId: "r1", supplierId: nil, supplierName: nil,
            status: .pending, items: [], totalAmount: 0, orderSource: "AI_PREDICTED",
            createdAt: "", updatedAt: "", estimatedDelivery: nil, qrCode: nil
        )
        #expect(order.isAiGenerated == true)
    }

    @Test func order_isAiGenerated_false() {
        let order = Order(
            id: "o1", retailerId: "r1", supplierId: nil, supplierName: nil,
            status: .pending, items: [], totalAmount: 0, orderSource: "MANUAL",
            createdAt: "", updatedAt: "", estimatedDelivery: nil, qrCode: nil
        )
        #expect(order.isAiGenerated == false)
    }

    @Test func order_itemCount_sumsQuantities() {
        let items = [
            OrderLineItem(id: "l1", productId: "p1", productName: "A", variantId: "v1", variantSize: "1L", quantity: 3, unitPrice: 10, totalPrice: 30),
            OrderLineItem(id: "l2", productId: "p2", productName: "B", variantId: "v2", variantSize: "2L", quantity: 2, unitPrice: 5, totalPrice: 10),
        ]
        let order = Order(
            id: "o1", retailerId: "r1", supplierId: nil, supplierName: nil,
            status: .pending, items: items, totalAmount: 40, orderSource: "MANUAL",
            createdAt: "", updatedAt: "", estimatedDelivery: nil, qrCode: nil
        )
        #expect(order.itemCount == 5)
    }
}
