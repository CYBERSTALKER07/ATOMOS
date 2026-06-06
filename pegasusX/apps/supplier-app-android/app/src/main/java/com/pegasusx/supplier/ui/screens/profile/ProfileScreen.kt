package com.pegasusx.supplier.ui.screens.profile

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierProfile
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(api: SupplierApi) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var profile by remember { mutableStateOf<SupplierProfile?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val resp = api.getProfile()
                if (resp.isSuccessful) profile = resp.body()
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Profile") }) }) { padding ->
        Box(Modifier.padding(padding)) {
            when {
                loading -> SupplierLoadingState("Loading profile…", "Supplier account")
                error != null -> SupplierStatePane(
                    kind = SupplierStateKind.Error,
                    headline = "Profile unavailable",
                    body = error!!,
                )
                profile == null -> SupplierStatePane(
                    kind = SupplierStateKind.Empty,
                    headline = "No profile",
                    body = "",
                )
                else -> {
                    val p = profile!!
                    Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        Text(p.legalName, style = MaterialTheme.typography.headlineSmall)
                        Text(p.contactName)
                        Text(p.email)
                        Text(p.phone)
                        Text("${p.country} · ${p.currency}")
                        Text("Configured: ${p.isConfigured}")
                    }
                }
            }
        }
    }
}
