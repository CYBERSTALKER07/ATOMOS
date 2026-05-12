import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogScreen.kt"
with open(target, "r") as f:
    text = f.read()

# Make sure we import Favorites
if "import androidx.compose.material.icons.rounded.Favorite" not in text:
    old_imports = """import androidx.compose.material.icons.rounded.Inventory2"""
    new_imports = """import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.FavoriteBorder"""
    text = text.replace(old_imports, new_imports)


old_topbar = """        topBar = {
            TopAppBar(
                title = { Text(supplierName) },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        }"""
new_topbar = """        topBar = {
            TopAppBar(
                title = { Text(supplierName) },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.toggleFavorite(supplierId) }) {
                        Icon(
                            imageVector = if (uiState.isFavorite) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder,
                            contentDescription = "Favorite",
                            tint = if (uiState.isFavorite) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        }"""

text = text.replace(old_topbar, new_topbar)

with open(target, "w") as f:
    f.write(text)

