import os

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogViewModel.kt"
with open(target, "r") as f:
    text = f.read()

old_state = """data class SupplierCatalogUiState(
    val isLoading: Boolean = true,
    val products: List<Product> = emptyList(),
    val error: String? = null,
)"""

new_state = """data class SupplierCatalogUiState(
    val isLoading: Boolean = true,
    val products: List<Product> = emptyList(),
    val error: String? = null,
    val isFavorite: Boolean = false,
)"""

text = text.replace(old_state, new_state)

old_load = """    fun load(supplierId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val products = api.getCatalogProducts(supplierId = supplierId)
                _uiState.update { it.copy(isLoading = false, products = products) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, products = emptyList(), error = e.message) }
            }
        }
    }"""

new_load = """    fun load(supplierId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                // Actually, there's no single supplier GET to check favorite state...
                // But we can fetch mySuppliers and see if it's there
                val products = api.getCatalogProducts(supplierId = supplierId)
                val mySuppliers = api.getMySuppliers()
                val isFav = mySuppliers.any { it.id == supplierId }
                _uiState.update { it.copy(isLoading = false, products = products, isFavorite = isFav) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, products = emptyList(), error = e.message) }
            }
        }
    }

    fun toggleFavorite(supplierId: String) {
        viewModelScope.launch {
            try {
                val isCurrentlyFav = _uiState.value.isFavorite
                if (isCurrentlyFav) {
                    api.removeSupplier(supplierId)
                    _uiState.update { it.copy(isFavorite = false) }
                } else {
                    api.addSupplier(supplierId)
                    _uiState.update { it.copy(isFavorite = true) }
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }"""

text = text.replace(old_load, new_load)

with open(target, "w") as f:
    f.write(text)

print("Patched SupplierCatalogViewModel")
