package com.thelab.retailer.data.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ModelComputedPropsTest {

    // ── Product ──

    @Test
    fun product_defaultVariant_firstVariant() {
        val v1 = Variant("v1", "1L", "Single", 1, "1000ml", 3.49)
        val v2 = Variant("v2", "2L", "Twin", 2, "2000ml", 6.49)
        val p = Product(id = "p1", name = "Milk", variants = listOf(v1, v2))
        assertEquals(v1, p.defaultVariant)
    }

    @Test
    fun product_defaultVariant_nullWhenEmpty() {
        val p = Product(id = "p1", name = "Milk")
        assertNull(p.defaultVariant)
    }

    @Test
    fun product_displayPrice_fromVariant() {
        val v = Variant("v1", "1L", "Single", 1, "1000ml", 3490.0)
        val p = Product(id = "p1", name = "Test", variants = listOf(v))
        assertEquals("3,490", p.displayPrice)
    }

    @Test
    fun product_displayPrice_fromPrice_whenNoVariants() {
        val p = Product(id = "p1", name = "Test", price = 5000)
        assertEquals("5,000", p.displayPrice)
    }

    @Test
    fun product_displayPrice_dash_whenNoPriceAtAll() {
        val p = Product(id = "p1", name = "Test")
        assertEquals("—", p.displayPrice)
    }

    @Test
    fun product_merchandisingLabel_categoryName() {
        val p = Product(id = "p1", name = "Test", categoryName = "Dairy")
        assertEquals("Dairy", p.merchandisingLabel)
    }

    @Test
    fun product_merchandisingLabel_sellByBlock() {
        val p = Product(id = "p1", name = "Test", sellByBlock = true, unitsPerBlock = 12)
        assertEquals("12 units / block", p.merchandisingLabel)
    }

    @Test
    fun product_merchandisingLabel_null_whenNothing() {
        val p = Product(id = "p1", name = "Test")
        assertNull(p.merchandisingLabel)
    }

    // ── Supplier ──

    @Test
    fun supplier_initials_twoWords() {
        val s = Supplier(id = "s1", name = "Fresh Farms")
        assertEquals("FF", s.initials)
    }

    @Test
    fun supplier_initials_singleWord() {
        val s = Supplier(id = "s1", name = "Bakery")
        assertEquals("BA", s.initials)
    }

    @Test
    fun supplier_initials_threeWords() {
        val s = Supplier(id = "s1", name = "The Lab Industries")
        assertEquals("TL", s.initials)
    }

    @Test
    fun supplier_displayCategory_directCategory() {
        val s = Supplier(id = "s1", name = "Test", category = "Dairy")
        assertEquals("Dairy", s.displayCategory)
    }

    @Test
    fun supplier_displayCategory_singleOperating() {
        val s = Supplier(id = "s1", name = "Test", operatingCategoryNames = listOf("Produce"))
        assertEquals("Produce", s.displayCategory)
    }

    @Test
    fun supplier_displayCategory_twoOperating() {
        val s = Supplier(id = "s1", name = "Test", operatingCategoryNames = listOf("A", "B"))
        assertEquals("A • B", s.displayCategory)
    }

    @Test
    fun supplier_displayCategory_moreOperating() {
        val s = Supplier(id = "s1", name = "Test", operatingCategoryNames = listOf("A", "B", "C", "D"))
        assertEquals("A +3 more", s.displayCategory)
    }

    @Test
    fun supplier_displayCategory_null_whenNone() {
        val s = Supplier(id = "s1", name = "Test")
        assertNull(s.displayCategory)
    }

    @Test
    fun supplier_catalogSubtitle_withProducts() {
        val s = Supplier(id = "s1", name = "Test", productCount = 42)
        assertEquals("42 products", s.catalogSubtitle)
    }

    @Test
    fun supplier_catalogSubtitle_fallsBackToOrders() {
        val s = Supplier(id = "s1", name = "Test", orderCount = 15)
        assertEquals("15 orders", s.catalogSubtitle)
    }

    // ── Order ──

    @Test
    fun order_displayTotal_formatsCorrectly() {
        val o = Order(id = "o1", totalAmount = 22.45)
        assertEquals("$22.45", o.displayTotal)
    }

    @Test
    fun order_itemCount_sumsQuantities() {
        val items = listOf(
            OrderLineItem("l1", "p1", "A", "v1", "1L", 3, 10.0, 30.0),
            OrderLineItem("l2", "p2", "B", "v2", "2L", 2, 5.0, 10.0),
        )
        val o = Order(id = "o1", items = items, totalAmount = 40.0)
        assertEquals(5, o.itemCount)
    }

    @Test
    fun order_isAiGenerated_true() {
        val o = Order(id = "o1", orderSource = "AI_PREDICTED")
        assertTrue(o.isAiGenerated)
    }

    @Test
    fun order_isAiGenerated_false() {
        val o = Order(id = "o1", orderSource = "MANUAL")
        assertFalse(o.isAiGenerated)
    }

    // ── CartItem ──

    @Test
    fun cartItem_totalPrice() {
        val v = Variant("v1", "1L", "Single", 1, "1000ml", 10_000.0)
        val p = Product(id = "p1", name = "Milk", variants = listOf(v))
        val item = CartItem(id = "p1_v1", product = p, variant = v, quantity = 3)
        assertEquals(30_000.0, item.totalPrice, 0.01)
    }

    // ── DemandForecast ──

    @Test
    fun demandForecast_confidencePercent() {
        val f = DemandForecast(id = "f1", confidence = 0.89)
        assertEquals("89%", f.confidencePercent)
    }

    // ── MonthlyExpense ──

    @Test
    fun monthlyExpense_shortMonth_valid() {
        val e = MonthlyExpense("2026-03", 100_000)
        assertEquals("Mar", e.shortMonth)
    }

    @Test
    fun monthlyExpense_shortMonth_january() {
        val e = MonthlyExpense("2026-01", 50_000)
        assertEquals("Jan", e.shortMonth)
    }

    @Test
    fun monthlyExpense_shortMonth_december() {
        val e = MonthlyExpense("2026-12", 200_000)
        assertEquals("Dec", e.shortMonth)
    }

    @Test
    fun monthlyExpense_shortMonth_invalidFormat() {
        val e = MonthlyExpense("March", 100_000)
        assertEquals("March", e.shortMonth)
    }
}
