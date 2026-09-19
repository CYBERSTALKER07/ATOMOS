package com.pegasusx.retailer.ui.screens.analytics

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.MonthlyExpense
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.data.model.RetailerAnalytics
import com.pegasusx.retailer.data.model.RetailerDetailedAnalytics
import com.pegasusx.retailer.data.model.TopProductExpense
import com.pegasusx.retailer.data.model.TopSupplierExpense
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class AnalyticsLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class DailySpend(
    val dayLabel: String,   // "M", "T", "W", etc.
    val amount: Long,
)

data class AnalyticsUiState(
    val isLoading: Boolean = false,
    val analytics: RetailerAnalytics? = null,
    val detailed: RetailerDetailedAnalytics? = null,
    val selectedRange: String = "30D",
    val weeklySpend: List<DailySpend> = emptyList(),
    val weeklyBudgetUzs: Long = 10_000_000,    // budget goal line
    val weekLabel: String = "This week",
    val avgPerDayUzs: Long = 0,
    val totalWeekUzs: Long = 0,
    val daysOnBudget: Int = 0,
    val predictions: List<RetailerAIPrediction> = emptyList(),
    val correctingPredictionId: String? = null,
    val error: String? = null,
    val loadIssue: AnalyticsLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            AnalyticsLoadIssue.RESTRICTED -> "Analytics access is restricted for this account"
            AnalyticsLoadIssue.OFFLINE -> "Offline mode active. Showing latest analytics"
            AnalyticsLoadIssue.ERROR -> "Analytics sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class AnalyticsViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AnalyticsUiState())
    val uiState: StateFlow<AnalyticsUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val expensesDeferred = async { api.getRetailerExpenses() }
                val detailedDeferred = async {
                    val range = rangeToDays(_uiState.value.selectedRange)
                    val to = java.time.LocalDate.now()
                    val from = to.minusDays(range.toLong())
                    api.getRetailerDetailedAnalytics(from.toString(), to.toString())
                }
                val analytics = expensesDeferred.await()
                val detailed = detailedDeferred.await()
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        analytics = analytics,
                        detailed = detailed,
                        predictions = emptyList(),
                        error = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun setRange(range: String) {
        _uiState.update { it.copy(selectedRange = range) }
        refresh()
    }

    private fun rangeToDays(range: String): Int = when (range) {
        "7D" -> 7
        "14D" -> 14
        "30D" -> 30
        "90D" -> 90
        "6M" -> 180
        "1Y" -> 365
        else -> 30
    }

    private fun resolveLoadIssue(error: Exception): AnalyticsLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> AnalyticsLoadIssue.RESTRICTED
            error is IOException -> AnalyticsLoadIssue.OFFLINE
            else -> AnalyticsLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: AnalyticsLoadIssue): String {
        return when (issue) {
            AnalyticsLoadIssue.RESTRICTED -> "Analytics access is restricted for this account"
            AnalyticsLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AnalyticsLoadIssue.ERROR -> error.message ?: "Analytics request failed"
        }
    }
}
