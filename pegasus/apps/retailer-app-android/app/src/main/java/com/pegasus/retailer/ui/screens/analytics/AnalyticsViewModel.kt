package com.pegasus.retailer.ui.screens.analytics

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.LabApi
import com.pegasus.retailer.data.model.MonthlyExpense
import com.pegasus.retailer.data.model.RetailerAnalytics
import com.pegasus.retailer.data.model.RetailerDetailedAnalytics
import com.pegasus.retailer.data.model.TopProductExpense
import com.pegasus.retailer.data.model.TopSupplierExpense
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

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
)

@HiltViewModel
class AnalyticsViewModel @Inject constructor(
    private val api: LabApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AnalyticsUiState())
    val uiState: StateFlow<AnalyticsUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
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
                _uiState.update { it.copy(isLoading = false, analytics = analytics, detailed = detailed) }
            } catch (_: Exception) {
                _uiState.update { it.copy(isLoading = false) }
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
}
