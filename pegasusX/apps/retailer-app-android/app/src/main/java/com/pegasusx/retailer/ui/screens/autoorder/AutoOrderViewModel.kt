package com.pegasusx.retailer.ui.screens.autoorder

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.AutoOrderRun
import com.pegasusx.retailer.data.model.AutoOrderRunsResponse
import com.pegasusx.retailer.data.model.AutoOrderSettings
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.RetailerReorderSuggestion
import com.pegasusx.retailer.data.model.UpdateGlobalSettingsRequest
import com.pegasusx.retailer.data.model.UpdateSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement
import retrofit2.HttpException
import javax.inject.Inject

enum class AutoOrderLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

/** Represents which entity the pending enable/history-choice dialog is for. */
sealed class EnableTarget {
    object Global : EnableTarget()
    data class Supplier(val id: String) : EnableTarget()
    data class Category(val id: String) : EnableTarget()
    data class Product(val id: String) : EnableTarget()
    data class Variant(val id: String) : EnableTarget()
}

data class AutoOrderUiState(
    val isLoading: Boolean = true,
    val settings: AutoOrderSettings? = null,
    val forecasts: List<DemandForecast> = emptyList(),
    val reorderSuggestions: List<RetailerReorderSuggestion> = emptyList(),
    val globalEnabled: Boolean = false,
    /** Non-null when the "Use past history or start fresh?" dialog should be shown. */
    val pendingEnableTarget: EnableTarget? = null,
    val error: String? = null,
    val loadIssue: AutoOrderLoadIssue? = null,
    val runs: List<AutoOrderRun> = emptyList(),
    val runsLoading: Boolean = false,
    val running: Boolean = false,
    val runningMode: String? = null,
    val lastRun: AutoOrderRun? = null,
    val runBanner: String? = null,
    val placeConfirmOpen: Boolean = false,
) {
    val syncMessage: String?
        get() = when {
            runBanner != null -> runBanner
            loadIssue == AutoOrderLoadIssue.RESTRICTED -> "Auto-order access is restricted for this account"
            loadIssue == AutoOrderLoadIssue.OFFLINE -> "Offline mode active. Showing latest auto-order settings"
            loadIssue == AutoOrderLoadIssue.ERROR -> "Auto-order sync degraded. Retry is available"
            else -> null
        }
}

@HiltViewModel
class AutoOrderViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: com.pegasusx.retailer.data.local.TokenManager,
) : ViewModel() {

    private val json = Json { ignoreUnknownKeys = true }

    private val _uiState = MutableStateFlow(AutoOrderUiState())
    val uiState: StateFlow<AutoOrderUiState> = _uiState.asStateFlow()

    init {
        loadAll()
    }

    fun loadAll() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            var nextIssue: AutoOrderLoadIssue? = null
            var nextError: String? = null
            var nextSettings: AutoOrderSettings? = null

            try {
                nextSettings = api.getAutoOrderSettings()
            } catch (e: Exception) {
                nextIssue = resolveLoadIssue(e)
                nextError = resolveErrorMessage(e, nextIssue)
            }

            val rid = tokenManager.getUserId().orEmpty()
            val nextForecasts = try {
                api.getPredictions(rid)
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
                _uiState.value.forecasts
            }

            val nextRuns = loadRunsInternal()
            val nextSuggestions = loadSuggestionsInternal()

            _uiState.update {
                val effectiveSettings = nextSettings ?: it.settings
                it.copy(
                    isLoading = false,
                    settings = effectiveSettings,
                    forecasts = nextForecasts,
                    reorderSuggestions = nextSuggestions,
                    globalEnabled = effectiveSettings?.globalEnabled ?: it.globalEnabled,
                    error = nextError,
                    loadIssue = nextIssue,
                    runs = nextRuns,
                    runsLoading = false,
                )
            }
        }
    }

    fun openPlaceConfirm() {
        _uiState.update { it.copy(placeConfirmOpen = true) }
    }

    fun dismissPlaceConfirm() {
        _uiState.update { it.copy(placeConfirmOpen = false) }
    }

    fun runAutoOrderNow() = runAutoOrder(mode = "draft")

    fun runAutoOrderPlace() = runAutoOrder(mode = "place")

    fun runAutoOrder(mode: String) {
        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    running = true,
                    runningMode = mode,
                    runBanner = null,
                    error = null,
                    placeConfirmOpen = false,
                )
            }
            try {
                val element = api.runAutoOrder(mode = mode)
                val run = json.decodeFromJsonElement(AutoOrderRun.serializer(), element)
                val banner = when {
                    mode == "place" && run.placedLines > 0 ->
                        "Place run: ${run.placedLines} line(s) in ${run.placedOrders.size} order(s)" +
                            (run.message?.let { " — $it" } ?: "")
                    mode == "place" ->
                        "Place run ${run.status}${run.message?.let { ": $it" } ?: ""}"
                    run.status == "OK" || run.status == "PARTIAL" ->
                        "Draft run complete: ${run.draftLines} line(s)" +
                            (run.message?.let { " — $it" } ?: "")
                    else ->
                        "Run ${run.status}${run.message?.let { ": $it" } ?: ""}"
                }
                val runs = loadRunsInternal()
                val suggestions = loadSuggestionsInternal()
                _uiState.update {
                    it.copy(
                        running = false,
                        runningMode = null,
                        lastRun = run,
                        runBanner = banner,
                        runs = runs,
                        reorderSuggestions = suggestions,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                val msg = resolvePlaceErrorMessage(e) ?: resolveErrorMessage(e, issue)
                _uiState.update {
                    it.copy(
                        running = false,
                        runningMode = null,
                        error = msg,
                        loadIssue = issue,
                        runBanner = msg,
                    )
                }
            }
        }
    }

    private fun resolvePlaceErrorMessage(e: Exception): String? {
        if (e !is HttpException) return null
        val body = try {
            e.response()?.errorBody()?.string().orEmpty()
        } catch (_: Exception) {
            ""
        }
        return when {
            body.contains("place_requires_manager") ->
                "Place requires OWNER, ADMIN, or MANAGER"
            body.contains("place_disabled") || body.contains("AUTO_ORDER_PLACE") ->
                "Place is disabled on server (AUTO_ORDER_PLACE_ENABLED)"
            body.contains("retailer_geo_missing") ->
                "Set primary location geo (lat/lng + H3) before place"
            body.contains("forbidden") ->
                "Missing order.place permission"
            else -> null
        }
    }

    private suspend fun loadRunsInternal(): List<AutoOrderRun> {
        return try {
            val element = api.getAutoOrderRuns()
            val parsed = json.decodeFromJsonElement(AutoOrderRunsResponse.serializer(), element)
            parsed.items
        } catch (_: Exception) {
            _uiState.value.runs
        }
    }

    private suspend fun loadSuggestionsInternal(): List<RetailerReorderSuggestion> {
        return try {
            api.getReorderSuggestions().items
        } catch (_: Exception) {
            _uiState.value.reorderSuggestions
        }
    }

    /**
     * Called when any entity toggle is switched ON.
     * If the entity has order history, show the "Continue/Fresh" dialog.
     * If no history exists, silently enable with useHistory=false (start fresh).
     */
    fun onToggle(target: EnableTarget, enabled: Boolean) {
        if (!enabled) {
            disableTarget(target)
            return
        }
        val hasHistory = entityHasHistory(target)
        if (hasHistory) {
            _uiState.update { it.copy(pendingEnableTarget = target) }
        } else {
            // No history — start fresh without prompting
            enableTarget(target, useHistory = false)
        }
    }

    /** Called from the dialog when the user picks "Use history" or "Start fresh". */
    fun confirmEnable(useHistory: Boolean) {
        val target = _uiState.value.pendingEnableTarget ?: return
        _uiState.update { it.copy(pendingEnableTarget = null) }
        enableTarget(target, useHistory)
    }

    fun dismissEnableDialog() {
        _uiState.update { it.copy(pendingEnableTarget = null) }
    }

    // ── Legacy entry-points kept for backward-compat call-sites ──────────────

    fun onGlobalToggle(enabled: Boolean) = onToggle(EnableTarget.Global, enabled)

    fun toggleSupplier(supplierId: String, enabled: Boolean) =
        onToggle(EnableTarget.Supplier(supplierId), enabled)

    fun toggleCategory(categoryId: String, enabled: Boolean) =
        onToggle(EnableTarget.Category(categoryId), enabled)

    fun toggleProduct(productId: String, enabled: Boolean) =
        onToggle(EnableTarget.Product(productId), enabled)

    fun toggleVariant(skuId: String, enabled: Boolean) =
        onToggle(EnableTarget.Variant(skuId), enabled)

    // ── Private helpers ───────────────────────────────────────────────────────

    private fun entityHasHistory(target: EnableTarget): Boolean {
        val s = _uiState.value.settings ?: return false
        return when (target) {
            is EnableTarget.Global -> s.hasAnyHistory
            is EnableTarget.Supplier -> s.supplierOverrides.firstOrNull { it.supplierId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Category -> s.categoryOverrides.firstOrNull { it.categoryId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Product -> s.productOverrides.firstOrNull { it.productId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Variant -> s.variantOverrides.firstOrNull { it.skuId == target.id }?.hasHistory ?: s.hasAnyHistory
        }
    }

    private fun enableTarget(target: EnableTarget, useHistory: Boolean) {
        viewModelScope.launch {
            _uiState.update { it.copy(error = null) }
            try {
                when (target) {
                    is EnableTarget.Global -> {
                        _uiState.update { it.copy(globalEnabled = true) }
                        api.updateGlobalAutoOrder(
                            UpdateGlobalSettingsRequest(globalAutoOrderEnabled = true, useHistory = useHistory)
                        )
                    }
                    is EnableTarget.Supplier ->
                        api.updateSupplierAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Category ->
                        api.updateCategoryAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                }
                loadAll()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = false) }
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun disableTarget(target: EnableTarget) {
        viewModelScope.launch {
            _uiState.update { it.copy(error = null) }
            try {
                when (target) {
                    is EnableTarget.Global -> {
                        _uiState.update { it.copy(globalEnabled = false) }
                        api.updateGlobalAutoOrder(UpdateGlobalSettingsRequest(globalAutoOrderEnabled = false))
                    }
                    is EnableTarget.Supplier ->
                        api.updateSupplierAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Category ->
                        api.updateCategoryAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                }
                loadAll()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = true) }
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): AutoOrderLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> AutoOrderLoadIssue.RESTRICTED
            error is IOException -> AutoOrderLoadIssue.OFFLINE
            else -> AutoOrderLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: AutoOrderLoadIssue): String {
        return when (issue) {
            AutoOrderLoadIssue.RESTRICTED -> "Auto-order access is restricted for this account"
            AutoOrderLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AutoOrderLoadIssue.ERROR -> error.message ?: "Auto-order request failed"
        }
    }
}

