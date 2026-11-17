package com.pegasus.retailer

import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.RetailerUserIdProvider
import com.pegasusx.retailer.data.model.DisputeOpeningBalanceRequest
import com.pegasusx.retailer.data.model.MigrationOpeningBalancesResponse
import com.pegasusx.retailer.data.model.MigrationOpeningInvoiceDto
import com.pegasusx.retailer.data.model.MigrationOpeningInvoiceMutationResponse
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.RetailerInnLookupRequest
import com.pegasusx.retailer.data.model.RetailerInnLookupResponse
import com.pegasusx.retailer.testsupport.RecordingPegasusApiConfig
import com.pegasusx.retailer.testsupport.httpJsonException
import com.pegasusx.retailer.ui.screens.credit.CreditViewModel
import com.pegasusx.retailer.ui.screens.setup.SetupWizardViewModel
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import com.pegasusx.retailer.util.RetailerW3ContractHelper
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST

@OptIn(ExperimentalCoroutinesApi::class)
class MigrationParityTests {

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
    fun api_routes_order_origin_and_idempotency_keys_match_contract() {
        val methods = PegasusApi::class.java.methods
        assertEquals(
            "/v1/platform/inn/lookup",
            methods.first { it.name == "lookupInn" }.getAnnotation(POST::class.java)?.value,
        )
        assertEquals(
            "/v1/retailer/tax-identity",
            methods.first { it.name == "updateTaxIdentity" }.getAnnotation(PATCH::class.java)?.value,
        )
        assertEquals(
            "/v1/retailer/nasiya/opening-balances",
            methods.first { it.name == "listNasiyaOpeningBalances" }.getAnnotation(GET::class.java)?.value,
        )
        assertEquals(
            "/v1/retailer/ar-invoices/{invoiceId}/acknowledge",
            methods.first { it.name == "acknowledgeOpeningBalance" }.getAnnotation(POST::class.java)?.value,
        )
        assertEquals(
            "/v1/retailer/ar-invoices/{invoiceId}/dispute",
            methods.first { it.name == "disputeOpeningBalance" }.getAnnotation(POST::class.java)?.value,
        )

        val orderJson = """
            {
              "order_id": "ord-mig-1",
              "retailer_id": "ret-1",
              "supplier_id": "sup-1",
              "supplier_name": "Navruz Food",
              "state": "CONFIRMED",
              "items": [],
              "amount": 150000,
              "currency": "UZS",
              "order_source": "MIGRATION"
            }
        """.trimIndent()
        val decodedOrder = OrderLineContractHelper.lenientJson.decodeFromString<Order>(orderJson)
        assertEquals("MIGRATION", decodedOrder.orderSource)
        assertTrue(decodedOrder.isMigrationOrigin)

        assertEquals("nasiya-opening-ack:inv-9", RetailerIdempotencyKeys.nasiyaAcknowledge("inv-9"))
        assertEquals(
            RetailerIdempotencyKeys.nasiyaDispute("inv-9", "wrong qty"),
            RetailerIdempotencyKeys.nasiyaDispute("inv-9", "wrong qty"),
        )
        assertNotEquals(
            RetailerIdempotencyKeys.nasiyaDispute("inv-9", "wrong qty"),
            RetailerIdempotencyKeys.nasiyaDispute("inv-9", "other"),
        )
    }

    @Test
    fun setupWizardViewModel_fails_closed_on_unavailable_allows_informal_and_handles_409_join_request() = runTest {
        val base = RecordingPegasusApiConfig().build()
        var setupCalls = 0
        val recordedPayloads = mutableListOf<Pair<Map<String, Any>, String?>>()
        var lookupMode = "UNAVAILABLE"
        var setupConflict = false

        val fakeApi = object : PegasusApi by base {
            override suspend fun lookupInn(body: RetailerInnLookupRequest): RetailerInnLookupResponse {
                return when (lookupMode) {
                    "FOUND" -> RetailerInnLookupResponse(
                        status = "FOUND",
                        normalizedInn = body.inn,
                        entityType = "OOO",
                        legalName = "Baraka Savdo",
                        address = "Tashkent",
                        existingTenantId = if (setupConflict) "ret-existing-1" else null,
                        existingJoinEligible = setupConflict,
                    )
                    else -> RetailerInnLookupResponse(status = "UNAVAILABLE", normalizedInn = body.inn)
                }
            }

            override suspend fun setupRetailer(
                body: Map<String, @JvmSuppressWildcards Any>,
                idempotencyKey: String?,
            ): JsonElement {
                setupCalls += 1
                recordedPayloads.add(body to idempotencyKey)
                if (setupConflict) {
                    throw httpJsonException(
                        409,
                        """{"error":"join_request_pending","request_id":"jr-77","target_tenant_id":"ret-existing-1","legal_name":"Baraka Savdo","normalized_inn":"301234567"}""",
                    )
                }
                return JsonObject(mapOf("id" to JsonPrimitive("ret-1")))
            }
        }

        // 1. INN UNAVAILABLE fails closed
        val unavailableVm = SetupWizardViewModel(fakeApi, RetailerUserIdProvider { "ret-u1" })
        unavailableVm.setTaxId("301234567")
        assertFalse(unavailableVm.canAdvanceFromTax())
        assertNotNull(unavailableVm.uiState.value.innError)
        assertEquals(0, setupCalls)

        // 2. Informal corner shop skips INN check and persists informal_shop=true
        val informalVm = SetupWizardViewModel(fakeApi, RetailerUserIdProvider { "ret-inf-1" })
        informalVm.setInformalShop(true)
        assertTrue(informalVm.canAdvanceFromTax())
        informalVm.setBillingAddress("Chorsu 12")
        informalVm.setShippingAddress("Chorsu 12")
        informalVm.setCity("Tashkent")
        assertTrue(informalVm.submitSetup())
        assertTrue(informalVm.uiState.value.setupCompleted)
        assertEquals("", recordedPayloads.last().first["tax_id"])
        assertEquals(true, recordedPayloads.last().first["informal_shop"])
        assertEquals("retailer-setup:ret-inf-1", recordedPayloads.last().second)

        // 3. Duplicate INN 409 join_request_pending does not complete setup and populates conflict
        lookupMode = "FOUND"
        setupConflict = true
        val joinVm = SetupWizardViewModel(fakeApi, RetailerUserIdProvider { "ret-join-1" })
        joinVm.setTaxId("301234567")
        assertTrue(joinVm.canAdvanceFromTax())
        joinVm.setBillingAddress("Amir Temur 10")
        joinVm.setShippingAddress("Amir Temur 10")
        joinVm.setCity("Tashkent")
        assertFalse(joinVm.submitSetup())
        assertFalse(joinVm.uiState.value.setupCompleted)
        assertTrue(joinVm.uiState.value.joinRequestSent)
        assertEquals("jr-77", joinVm.uiState.value.joinRequestConflict?.requestId)

        val parsed = RetailerW3ContractHelper.parseJoinRequestConflict(
            httpJsonException(
                409,
                """{"error":"join_request_pending","request_id":"jr-99","legal_name":"X"}""",
            ),
        )
        assertEquals("jr-99", parsed?.requestId)
    }

    @Test
    fun creditViewModel_loads_nasiya_opening_balances_and_confirms_or_disputes() = runTest {
        val base = RecordingPegasusApiConfig().build()
        val ackCalls = mutableListOf<Pair<String, String>>()
        val disputeCalls = mutableListOf<Triple<String, String, String>>()

        val fakeApi = object : PegasusApi by base {
            override suspend fun getCreditRelationships(): JsonElement {
                return OrderLineContractHelper.lenientJson.parseToJsonElement("""{"relationships":[]}""")
            }

            override suspend fun getArInvoices(status: String, limit: Int): JsonElement {
                return OrderLineContractHelper.lenientJson.parseToJsonElement("""{"invoices":[]}""")
            }

            override suspend fun listNasiyaOpeningBalances(): MigrationOpeningBalancesResponse {
                return MigrationOpeningBalancesResponse(
                    retailerId = "ret-1",
                    invoices = listOf(
                        MigrationOpeningInvoiceDto(
                            invoiceId = "inv-open-1",
                            supplierId = "sup-1",
                            supplierName = "Navruz Food",
                            externalId = "1C-AR-100",
                            principalMinor = 250_000_000L,
                            balanceMinor = 250_000_000L,
                            status = "PENDING_ACK",
                            source = "MIGRATION",
                            dunningFrozen = true,
                        ),
                        MigrationOpeningInvoiceDto(
                            invoiceId = "inv-open-2",
                            supplierId = "sup-2",
                            supplierName = "Orient Trade",
                            externalId = "1C-AR-200",
                            principalMinor = 120_000_000L,
                            balanceMinor = 120_000_000L,
                            status = "PENDING_ACK",
                            source = "MIGRATION",
                            dunningFrozen = true,
                        ),
                    ),
                )
            }

            override suspend fun acknowledgeOpeningBalance(
                invoiceId: String,
                body: Map<String, String>,
                idempotencyKey: String,
            ): MigrationOpeningInvoiceMutationResponse {
                ackCalls.add(invoiceId to idempotencyKey)
                return MigrationOpeningInvoiceMutationResponse(
                    retailerId = "ret-1",
                    invoice = MigrationOpeningInvoiceDto(
                        invoiceId = invoiceId,
                        supplierId = "sup-1",
                        supplierName = "Navruz Food",
                        balanceMinor = 250_000_000L,
                        status = "ACKNOWLEDGED",
                        dunningFrozen = false,
                    ),
                )
            }

            override suspend fun disputeOpeningBalance(
                invoiceId: String,
                body: DisputeOpeningBalanceRequest,
                idempotencyKey: String,
            ): MigrationOpeningInvoiceMutationResponse {
                disputeCalls.add(Triple(invoiceId, body.reason, idempotencyKey))
                return MigrationOpeningInvoiceMutationResponse(
                    retailerId = "ret-1",
                    invoice = MigrationOpeningInvoiceDto(
                        invoiceId = invoiceId,
                        supplierId = "sup-2",
                        supplierName = "Orient Trade",
                        balanceMinor = 120_000_000L,
                        status = "DISPUTED",
                        dunningFrozen = true,
                        disputeReason = body.reason,
                    ),
                )
            }
        }

        val vm = CreditViewModel(fakeApi)
        vm.refreshAll()

        assertEquals(2, vm.uiState.value.openingBalances.size)
        assertTrue(vm.uiState.value.openingBalances[0].dunningFrozen)

        assertTrue(vm.acknowledgeOpeningBalance("inv-open-1"))
        assertEquals(1, ackCalls.size)
        assertEquals("nasiya-opening-ack:inv-open-1", ackCalls[0].second)
        val confirmed = vm.uiState.value.openingBalances.first { it.invoiceId == "inv-open-1" }
        assertTrue(confirmed.isAcknowledged)
        assertFalse(confirmed.dunningFrozen)

        assertFalse(vm.disputeOpeningBalance("inv-open-2", "   "))
        assertTrue(disputeCalls.isEmpty())

        assertTrue(vm.disputeOpeningBalance("inv-open-2", "Paid cash in October"))
        assertEquals(1, disputeCalls.size)
        assertEquals("Paid cash in October", disputeCalls[0].second)
        assertEquals(
            RetailerIdempotencyKeys.nasiyaDispute("inv-open-2", "Paid cash in October"),
            disputeCalls[0].third,
        )
        val disputed = vm.uiState.value.openingBalances.first { it.invoiceId == "inv-open-2" }
        assertTrue(disputed.isDisputed)
        assertEquals("Paid cash in October", disputed.disputeReason)
    }

    @Test
    fun teamViewModel_lists_approves_and_rejects_retailer_join_requests() = runTest {
        val base = RecordingPegasusApiConfig().build()
        val approved = mutableListOf<Triple<String, String?, String>>()
        val rejected = mutableListOf<Triple<String, String?, String>>()
        val req1 = com.pegasusx.retailer.data.model.RetailerJoinRequestDto(
            requestId = "jr-and-1",
            requesterPhone = "+998901112233",
            requesterName = "Dilshod Store",
            roleSought = "RETAILER",
            status = "PENDING",
        )
        val req2 = com.pegasusx.retailer.data.model.RetailerJoinRequestDto(
            requestId = "jr-and-2",
            requesterPhone = "+998904445566",
            requesterName = "Unknown Caller",
            roleSought = "CASHIER",
            status = "PENDING",
        )
        val fakeApi = object : PegasusApi by base {
            override suspend fun listRetailerJoinRequests(status: String): com.pegasusx.retailer.data.model.RetailerJoinRequestsResponse {
                return com.pegasusx.retailer.data.model.RetailerJoinRequestsResponse(
                    retailerId = "ret-1",
                    requests = listOf(req1, req2),
                )
            }

            override suspend fun approveRetailerJoinRequest(
                requestId: String,
                body: com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionRequest,
                idempotencyKey: String,
            ): com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionResponse {
                approved.add(Triple(requestId, body.reason, idempotencyKey))
                return com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionResponse(
                    retailerId = "ret-1",
                    request = req1.copy(status = "APPROVED"),
                )
            }

            override suspend fun rejectRetailerJoinRequest(
                requestId: String,
                body: com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionRequest,
                idempotencyKey: String,
            ): com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionResponse {
                rejected.add(Triple(requestId, body.reason, idempotencyKey))
                return com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionResponse(
                    retailerId = "ret-1",
                    request = req2.copy(status = "REJECTED"),
                )
            }
        }

        val vm = com.pegasusx.retailer.ui.screens.settings.TeamViewModel(fakeApi)
        val loaded = vm.loadJoinRequests()
        assertEquals(2, loaded.size)
        assertEquals(2, vm.joinRequests.value.size)

        assertTrue(vm.approveJoinRequest("jr-and-1"))
        assertEquals(1, approved.size)
        assertEquals("retailer-join-decide:jr-and-1:approve", approved[0].third)
        assertEquals(listOf("jr-and-2"), vm.joinRequests.value.map { it.requestId })

        assertTrue(vm.rejectJoinRequest("jr-and-2", "Unknown"))
        assertEquals(1, rejected.size)
        assertEquals("retailer-join-decide:jr-and-2:reject", rejected[0].third)
        assertTrue(vm.joinRequests.value.isEmpty())
    }
}
