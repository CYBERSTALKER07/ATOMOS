package com.pegasus.retailer.ui.screens.catalog

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.Supplier
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class CategorySuppliersLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class CategorySuppliersUiState(
    val isLoading: Boolean = true,
    val suppliers: List<Supplier> = emptyList(),
    val error: String? = null,
    val loadIssue: CategorySuppliersLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            CategorySuppliersLoadIssue.RESTRICTED -> "Category suppliers access is restricted for this account"
            CategorySuppliersLoadIssue.OFFLINE -> "Offline mode active. Showing latest suppliers"
            CategorySuppliersLoadIssue.ERROR -> "Category suppliers sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class CategorySuppliersViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CategorySuppliersUiState())
    val uiState: StateFlow<CategorySuppliersUiState> = _uiState.asStateFlow()

    fun load(categoryId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null, loadIssue = null) }
            try {
                val suppliers = api.getCategorySuppliers(categoryId)
                _uiState.update { it.copy(isLoading = false, suppliers = suppliers, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue, "Could not load suppliers for this category"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): CategorySuppliersLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> CategorySuppliersLoadIssue.RESTRICTED
            error is IOException -> CategorySuppliersLoadIssue.OFFLINE
            else -> CategorySuppliersLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: CategorySuppliersLoadIssue, fallback: String): String {
        return when (issue) {
            CategorySuppliersLoadIssue.RESTRICTED -> "Category suppliers access is restricted for this account"
            CategorySuppliersLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            CategorySuppliersLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}