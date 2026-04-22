import Testing
@testable import reatilerapp

struct CartManagerTests {

    // Helper to create a product/variant pair
    private func makeProduct(id: String = "p1", name: String = "Test Product") -> Product {
        Product(
            id: id, name: name, description: "", nutrition: "",
            imageURL: nil, variants: [makeVariant()],
            supplierID: nil, supplierName: nil, supplierCategory: nil,
            categoryID: nil, categoryName: nil, sellByBlock: false,
            unitsPerBlock: nil, price: nil
        )
    }

    private func makeVariant(id: String = "v1", price: Double = 10_000) -> Variant {
        Variant(id: id, size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1000ml", price: price)
    }

    // MARK: - Initial state

    @Test func initialState_isEmpty() {
        let cart = CartManager()
        #expect(cart.isEmpty == true)
        #expect(cart.totalItems == 0)
        #expect(cart.totalPrice == 0)
    }

    // MARK: - Add

    @Test func add_singleItem() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        #expect(cart.items.count == 1)
        #expect(cart.totalItems == 1)
        #expect(cart.isEmpty == false)
    }

    @Test func add_incrementsExisting() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        cart.add(product: product, variant: variant)
        #expect(cart.items.count == 1)
        #expect(cart.totalItems == 2)
    }

    @Test func add_differentVariants_separateItems() {
        let cart = CartManager()
        let product = makeProduct()
        let v1 = makeVariant(id: "v1", price: 10_000)
        let v2 = makeVariant(id: "v2", price: 20_000)
        cart.add(product: product, variant: v1)
        cart.add(product: product, variant: v2)
        #expect(cart.items.count == 2)
    }

    @Test func add_customQuantity() {
        let cart = CartManager()
        cart.add(product: makeProduct(), variant: makeVariant(), quantity: 5)
        #expect(cart.totalItems == 5)
    }

    // MARK: - Remove

    @Test func remove_byItemId() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.remove(itemId: itemId)
        #expect(cart.isEmpty == true)
    }

    @Test func remove_nonexistentId_noop() {
        let cart = CartManager()
        cart.add(product: makeProduct(), variant: makeVariant())
        cart.remove(itemId: "nonexistent")
        #expect(cart.items.count == 1)
    }

    // MARK: - Update quantity

    @Test func updateQuantity_setsNewValue() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.updateQuantity(itemId: itemId, quantity: 10)
        #expect(cart.totalItems == 10)
    }

    @Test func updateQuantity_zeroRemovesItem() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.updateQuantity(itemId: itemId, quantity: 0)
        #expect(cart.isEmpty == true)
    }

    @Test func updateQuantity_negativeRemovesItem() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.updateQuantity(itemId: itemId, quantity: -1)
        #expect(cart.isEmpty == true)
    }

    // MARK: - Increment / Decrement

    @Test func increment_addsOne() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.increment(itemId: itemId)
        #expect(cart.totalItems == 2)
    }

    @Test func decrement_removesOne() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant, quantity: 3)
        let itemId = "\(product.id)-\(variant.id)"
        cart.decrement(itemId: itemId)
        #expect(cart.totalItems == 2)
    }

    @Test func decrement_lastItem_removesFromCart() {
        let cart = CartManager()
        let product = makeProduct()
        let variant = makeVariant()
        cart.add(product: product, variant: variant)
        let itemId = "\(product.id)-\(variant.id)"
        cart.decrement(itemId: itemId)
        #expect(cart.isEmpty == true)
    }

    // MARK: - Clear

    @Test func clear_removesAllItems() {
        let cart = CartManager()
        cart.add(product: makeProduct(id: "p1"), variant: makeVariant(id: "v1"))
        cart.add(product: makeProduct(id: "p2"), variant: makeVariant(id: "v2"))
        cart.clear()
        #expect(cart.isEmpty == true)
    }

    // MARK: - Computed properties

    @Test func totalPrice_sumsCorrectly() {
        let cart = CartManager()
        let v1 = makeVariant(id: "v1", price: 10_000)
        let v2 = makeVariant(id: "v2", price: 25_000)
        let p1 = makeProduct(id: "p1")
        let p2 = makeProduct(id: "p2")
        cart.add(product: p1, variant: v1, quantity: 3)  // 30_000
        cart.add(product: p2, variant: v2, quantity: 2)  // 50_000
        #expect(cart.totalPrice == 80_000)
    }

    @Test func displayTotal_formatsAmount() {
        let cart = CartManager()
        let variant = makeVariant(price: 50_000)
        cart.add(product: makeProduct(), variant: variant, quantity: 2)
        // 100_000 formatted
        #expect(cart.displayTotal.contains("100"))
    }

    // MARK: - Checkout payload

    @Test func buildCheckoutPayload_correctStructure() {
        let cart = CartManager()
        let product = makeProduct(id: "sku-abc")
        let variant = makeVariant(id: "v1", price: 15_000)
        cart.add(product: product, variant: variant, quantity: 3)

        let payload = cart.buildCheckoutPayload(
            retailerId: "r123",
            paymentGateway: "CLICK",
            latitude: 41.3,
            longitude: 69.3
        )
        #expect(payload.retailerId == "r123")
        #expect(payload.paymentGateway == "CLICK")
        #expect(payload.items.count == 1)
        #expect(payload.items[0].quantity == 3)
    }
}
