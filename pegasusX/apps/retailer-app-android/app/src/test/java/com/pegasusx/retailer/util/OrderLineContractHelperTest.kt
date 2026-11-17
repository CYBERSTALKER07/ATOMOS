package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderLineItem
import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.ProcurementOrderItem
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.testsupport.httpJsonException
import kotlinx.serialization.encodeToString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class OrderLineContractHelperTest {

    @Test
    fun snapToPackMultiple_handlesNearestFloorCeilAndBoundaries() {
        assertEquals(0, OrderLineContractHelper.snapToPackMultiple(0, 6, PackSnapMode.NEAREST))
        assertEquals(6, OrderLineContractHelper.snapToPackMultiple(6, 6, PackSnapMode.NEAREST))
        assertEquals(6, OrderLineContractHelper.snapToPackMultiple(2, 6, PackSnapMode.NEAREST))
        assertEquals(12, OrderLineContractHelper.snapToPackMultiple(9, 6, PackSnapMode.NEAREST)) // tie rounds up
        assertEquals(6, OrderLineContractHelper.snapToPackMultiple(9, 6, PackSnapMode.FLOOR))
        assertEquals(12, OrderLineContractHelper.snapToPackMultiple(7, 6, PackSnapMode.CEIL))
        assertEquals(5, OrderLineContractHelper.snapToPackMultiple(5, 1, PackSnapMode.NEAREST))
    }

    @Test
    fun effectivePackSize_prefersVariantPackCountElseSellByBlock() {
        val packVariant = Variant(
            id = "v1",
            size = "1L",
            pack = "Case 12",
            packCount = 12,
            weightPerUnit = "1000ml",
            price = 1_000L,
        )
        val prod1 = Product(id = "p1", name = "Water", variants = listOf(packVariant), unitsPerBlock = 6, sellByBlock = true)
        assertEquals(12, OrderLineContractHelper.effectivePackSize(prod1, packVariant))

        val singleVariant = Variant(
            id = "v2",
            size = "0.5L",
            pack = "Single",
            packCount = 1,
            weightPerUnit = "500ml",
            price = 1_000L,
        )
        val prod2 = Product(id = "p2", name = "Juice", variants = listOf(singleVariant), unitsPerBlock = 8, sellByBlock = true)
        assertEquals(8, OrderLineContractHelper.effectivePackSize(prod2, singleVariant))

        val prod3 = Product(id = "p3", name = "Bread", variants = listOf(singleVariant), sellByBlock = false)
        assertEquals(1, OrderLineContractHelper.effectivePackSize(prod3, singleVariant))
    }

    @Test
    fun formatPriceGapSom_formatsRoundedIntegerSomWithoutNegativeZero() {
        assertEquals("+1 250 so'm", OrderLineContractHelper.formatPriceGapSom(125_000L))
        assertEquals("-1 250 so'm", OrderLineContractHelper.formatPriceGapSom(-125_000L))
        assertEquals("+2 so'm", OrderLineContractHelper.formatPriceGapSom(150L))
        assertEquals("-2 so'm", OrderLineContractHelper.formatPriceGapSom(-150L))
        assertEquals("0 so'm", OrderLineContractHelper.formatPriceGapSom(0L))
        assertEquals("0 so'm", OrderLineContractHelper.formatPriceGapSom(-40L))
        assertEquals("0 so'm", OrderLineContractHelper.formatPriceGapSom(40L))
        assertEquals("+500 so'm", OrderLineContractHelper.formatPriceGapSom(50_000L, "UZS"))
    }

    @Test
    fun parsePackMultipleSuggestions_parsesHttpAndProblemDetailPayloads() {
        val nestedBody = """
            {
              "code": "PACK_MULTIPLE_REQUIRED",
              "message": "Quantities must match pack multiples",
              "details": [
                {
                  "sku_id": "sku-cola-12",
                  "product_name": "Cola 0.5L",
                  "pack_size": 12,
                  "requested": 14,
                  "suggested_floor": 12,
                  "suggested_ceil": 24,
                  "suggested_nearest": 12
                }
              ]
            }
        """.trimIndent()
        val fromHttp = OrderLineContractHelper.parsePackMultipleSuggestions(
            httpJsonException(422, nestedBody),
        )
        assertEquals(1, fromHttp.size)
        assertEquals("sku-cola-12", fromHttp[0].skuId)
        assertEquals(12, fromHttp[0].packSize)
        assertEquals(12, fromHttp[0].suggestedFloor)
        assertEquals(24, fromHttp[0].suggestedCeil)
        assertEquals(12, fromHttp[0].suggestedNearest)

        val problem = ProblemDetail(
            status = 422,
            code = "PACK_MULTIPLE_REQUIRED",
            detail = "Order in packs of 6",
            skuId = "sku-milk-6",
            packSize = 6,
            requested = 10,
            suggestedFloor = 6,
            suggestedCeil = 12,
            suggestedNearest = 12,
        )
        val fromProblem = OrderLineContractHelper.parsePackMultipleSuggestions(
            ProblemDetailException(problem),
        )
        assertEquals(1, fromProblem.size)
        assertEquals("sku-milk-6", fromProblem[0].skuId)
        assertEquals(6, fromProblem[0].suggestedFloor)
        assertEquals(12, fromProblem[0].suggestedCeil)
        assertEquals(12, fromProblem[0].suggestedNearest)
    }

    @Test
    fun serialization_preservesSubstitutePriceGapAndOptInFields() {
        val json = OrderLineContractHelper.lenientJson
        val orderJson = """
            {
              "order_id": "ord-99",
              "retailer_id": "ret-1",
              "state": "CONFIRMED",
              "amount": 500000,
              "auto_confirm_opt_in": true,
              "created_at": "2026-11-13T00:00:00Z",
              "updated_at": "2026-11-13T00:00:00Z",
              "items": [
                {
                  "line_item_id": "li-1",
                  "sku_id": "sku-sub",
                  "sku_name": "Milk 2.5%",
                  "quantity": 6,
                  "unit_price": 85000,
                  "total_price": 510000,
                  "must_have": false,
                  "allow_substitutes": true,
                  "substitute_sku": "sku-sub",
                  "substitute_name": "Milk 2.5% Alt",
                  "original_sku": "sku-orig",
                  "price_gap_minor": 150000
                }
              ]
            }
        """.trimIndent()
        val decodedOrder = json.decodeFromString<Order>(orderJson)
        assertTrue(decodedOrder.autoConfirmOptIn)
        val line: OrderLineItem = decodedOrder.items.single()
        assertFalse(line.mustHave)
        assertTrue(line.allowSubstitutes)
        assertEquals("sku-sub", line.substituteSku)
        assertEquals("Milk 2.5% Alt", line.substituteName)
        assertEquals("sku-orig", line.originalSku)
        assertEquals(150_000L, line.priceGapMinor)

        val checkoutReq = UnifiedCheckoutRequest(
            retailerId = "ret-1",
            paymentGateway = "CASH",
            items = listOf(
                CheckoutLineItem(
                    skuId = "sku-1",
                    quantity = 12,
                    unitPrice = 50_000L,
                    mustHave = true,
                    allowSubstitutes = false,
                ),
            ),
            autoConfirmOptIn = true,
        )
        val encodedCheckout = json.encodeToString(checkoutReq)
        assertTrue(encodedCheckout.contains("\"must_have\":true"))
        assertTrue(encodedCheckout.contains("\"allow_substitutes\":false"))
        assertTrue(encodedCheckout.contains("\"auto_confirm_opt_in\":true"))

        val procReq = ProcurementOrderRequest(
            retailerId = "ret-1",
            items = listOf(ProcurementOrderItem(productId = "p-1", quantity = 6, mustHave = true, allowSubstitutes = true)),
            autoConfirmOptIn = true,
        )
        val encodedProc = json.encodeToString(procReq)
        assertTrue(encodedProc.contains("\"must_have\":true"))
        assertTrue(encodedProc.contains("\"allow_substitutes\":true"))
        assertTrue(encodedProc.contains("\"auto_confirm_opt_in\":true"))
    }

    @Test
    fun assessPendingOrderError_preserves401And429AndFormatsWholeSom() {
        val unauthorized = OrderLineContractHelper.assessPendingOrderError(
            ProblemDetailException(ProblemDetail(status = 401, code = "unauthorized", detail = "Token expired")),
        )
        assertEquals(PendingOrderConflictKind.UNAUTHORIZED, unauthorized.kind)
        assertEquals("AUTH_REQUIRED", unauthorized.status)
        assertFalse(unauthorized.shouldDelete)
        assertFalse(unauthorized.retryable)

        val rateLimited = OrderLineContractHelper.assessPendingOrderError(
            ProblemDetailException(ProblemDetail(status = 429, code = "rate_limit_exceeded", detail = "Slow down")),
        )
        assertEquals(PendingOrderConflictKind.RATE_LIMITED, rateLimited.kind)
        assertFalse(rateLimited.shouldDelete)
        assertTrue(rateLimited.retryable)

        assertEquals("15 000 so'm", OrderLineContractHelper.formatWholeSom(1_500_000L))
        assertEquals("1 so'm", OrderLineContractHelper.formatWholeSom(50L))
        assertEquals("0 so'm", OrderLineContractHelper.formatWholeSom(0L))
        assertEquals("-125 so'm", OrderLineContractHelper.formatWholeSom(-12_540L))
    }
}
