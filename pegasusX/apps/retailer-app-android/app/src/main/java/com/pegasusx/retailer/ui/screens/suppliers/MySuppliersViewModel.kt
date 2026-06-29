package com.pegasusx.retailer.ui.screens.suppliers

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.Supplier
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class SuppliersLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class MySuppliersUiState(
    val isLoading: Boolean = false,
    val isRefreshing: Boolean = false,
    val isSearching: Boolean = false,
    val searchQuery: String = "",
    val suppliers: List<Supplier> = emptyList(),
    val searchResults: List<Supplier> = emptyList(),
    val error: String? = null,
    val loadIssue: SuppliersLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            SuppliersLoadIssue.RESTRICTED -> "Supplier list access is restricted for this account"
            SuppliersLoadIssue.OFFLINE -> "Offline mode active. Showing latest suppliers"
            SuppliersLoadIssue.ERROR -> "Supplier sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class MySuppliersViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(MySuppliersUiState())
    val uiState: StateFlow<MySuppliersUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update {
                val hasCached = it.suppliers.isNotEmpty()
                it.copy(
                    isLoading = !hasCached,
                    isRefreshing = hasCached,
                    error = null,
                )
            }

            try {
                val suppliers = api.getMySuppliers()
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        isRefreshing = false,
                        suppliers = suppliers,
                        error = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        isRefreshing = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun addSupplier(supplierId: String) {
        viewModelScope.launch {
            try {
                api.addSupplier(supplierId, RetailerIdempotencyKeys.supplierAdd(supplierId))
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun removeSupplier(supplierId: String) {
        viewModelScope.launch {
            try {
                api.removeSupplier(supplierId, RetailerIdempotencyKeys.supplierRemove(supplierId))
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun searchSuppliers(query: String) {
        val trimmed = query.trim()
        _uiState.update { it.copy(searchQuery = trimmed) }
        if (trimmed.length < 2) {
            _uiState.update { it.copy(searchResults = emptyList(), isSearching = false) }
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(isSearching = true, error = null) }
            try {
                val results = api.searchSuppliers(trimmed)
                val existing = _uiState.value.suppliers.map { it.id }.toSet()
                _uiState.update {
                    it.copy(
                        isSearching = false,
                        searchResults = results.filter { supplier -> supplier.id !in existing },
                        loadIssue = null,
                        error = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isSearching = false,
                        searchResults = emptyList(),
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): SuppliersLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> SuppliersLoadIssue.RESTRICTED
            error is IOException -> SuppliersLoadIssue.OFFLINE
            else -> SuppliersLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: SuppliersLoadIssue): String {
        return when (issue) {
            SuppliersLoadIssue.RESTRICTED -> "Supplier list access is restricted for this account"
            SuppliersLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            SuppliersLoadIssue.ERROR -> error.message ?: "Supplier list unavailable. Check your connection"
        }
    }
}
