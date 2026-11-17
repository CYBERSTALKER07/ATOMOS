package com.pegasusx.retailer.ui.screens

import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.ArInvoiceDto
import com.pegasusx.retailer.data.model.ArPromiseDto
import com.pegasusx.retailer.data.model.ArPromisesResponse
import com.pegasusx.retailer.data.model.ArStatementResponse
import com.pegasusx.retailer.data.model.CreateArPromiseRequest
import com.pegasusx.retailer.data.model.PayArFifoAllocationDto
import com.pegasusx.retailer.data.model.PayArFifoRequest
import com.pegasusx.retailer.data.model.PayArFifoResponse
import com.pegasusx.retailer.data.model.PayArInvoiceRequest
import com.pegasusx.retailer.testsupport.RecordingPegasusApiConfig
import com.pegasusx.retailer.testsupport.httpJsonException
import com.pegasusx.retailer.ui.screens.credit.CreditViewModel
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import com.pegasusx.retailer.util.RetailerW3ContractHelper
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.http.GET
import retrofit2.http.POST

@OptIn(ExperimentalCoroutinesApi::class)
class RetailerW3Tests {

    private val testDispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun credit_ui_never_displays_raw_tiyin_or_formatTiyinFixed() {
        val root = File(System.getProperty("user.dir") ?: ".")
        val creditScreenFile = listOf(
            File(root, "app/src/main/java/com/pegasusx/retailer/ui/screens/credit/CreditScreen.kt"),
            File(root, "src/main/java/com/pegasusx/retailer/ui/screens/credit/CreditScreen.kt"),
        ).first { it.exists() }
        val profileCardFile = listOf(
            File(root, "app/src/main/java/com/pegasusx/retailer/ui/screens/profile/components/CreditProfileCard.kt"),
            File(root, "src/main/java/com/pegasusx/retailer/ui/screens/profile/components/CreditProfileCard.kt"),
        ).first { it.exists() }

        val creditSrc = creditScreenFile.readText()
        val profileSrc = profileCardFile.readText()

        assertFalse(creditSrc.contains("formatTiyinFixed"))
        assertTrue(creditSrc.contains("OrderLineContractHelper.formatWholeSom"))
        assertFalse(profileSrc.contains("formatter.format"))
        assertTrue(profileSrc.contains("OrderLineContractHelper.formatWholeSom"))
    }

    @Test
    fun pegasusApi_annotations_match_locked_w3_routes() {
        val methods = PegasusApi::class.java.methods
        val getStatement = methods.first { it.name == "getArStatement" }
        assertEquals("/v1/retailer/ar/statement", getStatement.getAnnotation(GET::class.java)?.value)

        val payFifo = methods.first { it.name == "payArFifo" }
        assertEquals("/v1/retailer/ar/pay-fifo", payFifo.getAnnotation(POST::class.java)?.value)

        val payInvoice = methods.first { it.name == "payArInvoice" }
        assertEquals("/v1/retailer/ar/invoices/{id}/pay", payInvoice.getAnnotation(POST::class.java)?.value)

        val getPromises = methods.first { it.name == "getArPromises" }
        assertEquals("/v1/retailer/ar/promises", getPromises.getAnnotation(GET::class.java)?.value)

        val createPromise = methods.first { it.name == "createArPromise" }
        assertEquals("/v1/retailer/ar/promises", createPromise.getAnnotation(POST::class.java)?.value)
    }

    @Test
    fun helpers_sort_oldest_first_format_whole_som_and_parse_409_limit() {
        val sorted = RetailerW3ContractHelper.sortArInvoicesOldestFirst(
            listOf(
                ArInvoiceDto(invoiceId = "inv-new", dueAt = "2026-12-01T00:00:00Z", balanceMinor = 20_000_00L),
                ArInvoiceDto(invoiceId = "inv-old", dueAt = "2026-10-15T00:00:00Z", balanceMinor = 10_000_00L),
            ),
        )
        assertEquals(listOf("inv-old", "inv-new"), sorted.map { it.invoiceId })
        assertEquals(50_000_000L, RetailerW2ContractHelper.somInputToMinor("500000"))
        assertEquals("500 000 so'm", OrderLineContractHelper.formatWholeSom(50_000_000L))

        val http409 = httpJsonException(
            409,
            """{"code":"quarterly_promise_limit_reached","error":"quarterly_promise_limit_reached","quarter_key":"2026-Q4","message":"Already used"}""",
        )
        val parsed = RetailerW3ContractHelper.parseQuarterlyPromiseLimitError(http409)
        assertNotNull(parsed)
        assertEquals("quarterly_promise_limit_reached", parsed?.code)
        assertEquals("2026-Q4", parsed?.quarterKey)
    }

    @Test
    fun creditViewModel_loads_statement_submits_fifo_and_earmark_and_surfaces_409_banner() = runTest {
        val base = RecordingPegasusApiConfig().build()
        val recordedStatuses = mutableListOf<String>()
        val fifoCalls = mutableListOf<Pair<PayArFifoRequest, String>>()
        val invoicePayCalls = mutableListOf<Triple<String, PayArInvoiceRequest, String>>()
        var failPromise409 = true

        val fakeApi = object : PegasusApi by base {
            override suspend fun getCreditRelationships(): JsonElement {
                return OrderLineContractHelper.lenientJson.parseToJsonElement(
                    """
                    {
                      "relationships": [
                        {
                          "supplier_id": "sup-1",
                          "supplier_name": "Atlas Supply",
                          "credit_enabled": true,
                          "terms_days": 14,
                          "grace_period_days": 3,
                          "credit_limit_minor": 100000000,
                          "available_credit_minor": 40000000,
                          "current_balance_minor": 60000000,
                          "on_hold": false,
                          "prepay_only_locked": false
                        }
                      ]
                    }
                    """.trimIndent(),
                )
            }

            override suspend fun getArInvoices(status: String, limit: Int): JsonElement {
                recordedStatuses.add(status)
                return JsonObject(emptyMap())
            }

            override suspend fun getArStatement(supplierId: String?, limit: Int): ArStatementResponse {
                return ArStatementResponse(
                    supplierId = supplierId ?: "sup-1",
                    currency = "UZS",
                    totalOpenMinor = 60_000_000L,
                    totalDisputedMinor = 10_000_000L,
                    totalUndisputedMinor = 50_000_000L,
                    advanceBalanceMinor = 5_000_000L,
                    currentQuarterKey = "2026-Q4",
                    selfServePromiseUsed = false,
                    maxExtensionDays = 14,
                    invoices = listOf(
                        ArInvoiceDto(
                            invoiceId = "inv-2",
                            supplierId = "sup-1",
                            orderId = "ORD-2",
                            status = "OPEN",
                            balanceMinor = 40_000_000L,
                            dueAt = "2026-11-25T00:00:00Z",
                            agingBucket = "CURRENT",
                        ),
                        ArInvoiceDto(
                            invoiceId = "inv-1",
                            supplierId = "sup-1",
                            orderId = "ORD-1",
                            status = "PARTIAL",
                            balanceMinor = 20_000_000L,
                            disputedMinor = 10_000_000L,
                            undisputedMinor = 10_000_000L,
                            dueAt = "2026-10-05T00:00:00Z",
                            agingBucket = "15_30_DAYS",
                        ),
                    ),
                    promises = emptyList(),
                )
            }

            override suspend fun getArPromises(supplierId: String?): ArPromisesResponse {
                return ArPromisesResponse(promises = emptyList())
            }

            override suspend fun payArFifo(
                body: PayArFifoRequest,
                idempotencyKey: String,
            ): PayArFifoResponse {
                fifoCalls.add(body to idempotencyKey)
                return PayArFifoResponse(
                    paidMinor = body.amountMinor,
                    advanceAddedMinor = 2_000_000L,
                    advanceBalanceMinor = 7_000_000L,
                    allocations = listOf(
                        PayArFifoAllocationDto(
                            invoiceId = "inv-1",
                            appliedMinor = body.amountMinor - 2_000_000L,
                            remainingBalanceMinor = 0L,
                            status = "PAID",
                        ),
                    ),
                )
            }

            override suspend fun payArInvoice(
                invoiceId: String,
                body: PayArInvoiceRequest,
                idempotencyKey: String,
            ): JsonElement {
                invoicePayCalls.add(Triple(invoiceId, body, idempotencyKey))
                return JsonObject(emptyMap())
            }

            override suspend fun createArPromise(
                body: CreateArPromiseRequest,
                idempotencyKey: String,
            ): ArPromiseDto {
                if (failPromise409) {
                    throw httpJsonException(
                        409,
                        """{"code":"quarterly_promise_limit_reached","error":"quarterly_promise_limit_reached","quarter_key":"2026-Q4","message":"Limit reached"}""",
                    )
                }
                return ArPromiseDto(
                    promiseId = "prom-1",
                    supplierId = body.supplierId,
                    quarterKey = "2026-Q4",
                    dueAt = "2026-11-30T00:00:00Z",
                    promisedMinor = 10_000_000L,
                    status = "ACTIVE",
                )
            }
        }

        val vm = CreditViewModel(fakeApi)
        vm.refreshAll()

        val loaded = vm.uiState.value
        assertEquals("sup-1", loaded.selectedSupplierId)
        assertEquals(listOf("OPEN,PARTIAL"), recordedStatuses)
        assertEquals(listOf("inv-1", "inv-2"), loaded.invoices.map { it.invoiceId })
        assertEquals(5_000_000L, loaded.advanceBalanceMinor)
        assertEquals(10_000_000L, loaded.totalDisputedMinor)
        assertEquals(50_000_000L, loaded.totalUndisputedMinor)

        val paidOk = vm.submitPayFifo(
            supplierId = "sup-1",
            amountMinor = 30_000_000L,
            paymentMethod = "WALLET",
        )
        assertTrue(paidOk)
        assertEquals(1, fifoCalls.size)
        assertEquals(30_000_000L, fifoCalls[0].first.amountMinor)
        assertEquals("WALLET", fifoCalls[0].first.paymentMethod)
        assertEquals(36, fifoCalls[0].second.length)
        assertEquals(30_000_000L, vm.uiState.value.paySuccessPaidMinor)
        assertEquals(2_000_000L, vm.uiState.value.paySuccessAdvanceMinor)

        val invOk = vm.submitPayInvoice(
            invoiceId = "inv-1",
            amountMinor = 10_000_000L,
            paymentMethod = "CARD",
        )
        assertTrue(invOk)
        assertEquals(1, invoicePayCalls.size)
        assertEquals("inv-1", invoicePayCalls[0].first)
        assertEquals(10_000_000L, invoicePayCalls[0].second.amountMinor)

        val promOk = vm.submitPromiseExtension("sup-1", 7)
        assertFalse(promOk)
        assertEquals("2026-Q4", vm.uiState.value.quarterlyLimitError?.quarterKey)
    }
}
