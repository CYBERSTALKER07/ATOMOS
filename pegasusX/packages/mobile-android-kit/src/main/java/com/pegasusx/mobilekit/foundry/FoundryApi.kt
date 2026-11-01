package com.pegasusx.mobilekit.foundry

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface FoundryApi {

    @POST("/v1/foundry/ontology/lot-recall")
    suspend fun triggerLotRecall(@Body req: LotRecallRequest): Response<RecallBlastRadiusReport>

    @POST("/v1/foundry/kinetic/actions/execute")
    suspend fun executeAction(@Body req: ActionExecutionRequest): Response<ActionExecutionRecord>

    @POST("/v1/foundry/scenarios/{scenarioID}/patches")
    suspend fun applyScenarioPatch(
        @Path("scenarioID") scenarioID: String,
        @Body req: ScenarioPatchRequest,
    ): Response<Map<String, kotlinx.serialization.json.JsonElement>>

    @GET("/v1/foundry/scenarios/{scenarioID}/resolve/{cellID}/{objectType}/{objectID}")
    suspend fun resolveScenarioObject(
        @Path("scenarioID") scenarioID: String,
        @Path("cellID") cellID: String,
        @Path("objectType") objectType: String,
        @Path("objectID") objectID: String,
    ): Response<OntologyObjectDto>

    @POST("/v1/foundry/scenarios/{scenarioID}/commit")
    suspend fun commitScenario(@Path("scenarioID") scenarioID: String): Response<ScenarioCommitResult>

    @POST("/v1/foundry/cashier/sessions")
    suspend fun startCashierSession(@Body req: StartCashierSessionRequest): Response<CashierSessionView>

    @POST("/v1/foundry/cashier/sessions/{sessionID}/blind-count")
    suspend fun submitBlindCount(
        @Path("sessionID") sessionID: String,
        @Body req: BlindCountRequest,
    ): Response<CashierSessionView>

    @POST("/v1/foundry/cashier/reclassify-tender")
    suspend fun reclassifyTender(@Body req: RRNTenderReclassificationRequest): Response<CashierSessionView>

    @POST("/v1/foundry/cashier/vault/cit-handover")
    suspend fun executeCITHandover(@Body req: CITHandoverRequest): Response<DepotVaultLedgerEntry>

    @POST("/v1/foundry/fleet/dvir")
    suspend fun submitDVIR(@Body req: FleetDVIRInspection): Response<FleetDVIRInspection>

    @POST("/v1/foundry/fleet/gate-events")
    suspend fun recordGateEvent(@Body req: YardGateCheckEvent): Response<YardGateCheckEvent>

    @POST("/v1/foundry/fleet/mid-route-rescue")
    suspend fun executeMidRouteRescue(@Body req: MidRouteRescueRequest): Response<MidRouteRescueResult>

    @POST("/v1/foundry/fleet/predictive-reefer")
    suspend fun checkPredictiveReefer(@Body req: PredictiveReeferRequest): Response<PredictiveReeferResponse>

    @GET("/v1/foundry/fleet/vehicles/{vehicleID}/dispatch-check")
    suspend fun checkVehicleDispatch(@Path("vehicleID") vehicleID: String): Response<DispatchCheckResponse>

    @POST("/v1/foundry/guards/aml-check")
    suspend fun checkAML(@Body req: AMLCheckRequest): Response<AMLCheckResponse>

    @POST("/v1/foundry/ontology/search-around")
    suspend fun searchAround(@Body req: SearchAroundRequest): Response<SearchAroundResponse>
}
