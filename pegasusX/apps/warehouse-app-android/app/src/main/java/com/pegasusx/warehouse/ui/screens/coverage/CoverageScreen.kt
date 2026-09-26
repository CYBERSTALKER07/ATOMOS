package com.pegasusx.warehouse.ui.screens.coverage

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.R
import com.pegasusx.warehouse.data.model.WarehouseCoverageResponse
import com.pegasusx.warehouse.data.model.WarehouseSupplyFactoryResponse
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import androidx.compose.ui.res.stringResource

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CoverageScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var coverage by remember { mutableStateOf<WarehouseCoverageResponse?>(null) }
    var factory by remember { mutableStateOf<WarehouseSupplyFactoryResponse?>(null) }
    var factoryError by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            factoryError = null
            try {
                val cov = api.getOpsCoverage()
                if (cov.isSuccessful) {
                    coverage = cov.body()
                } else {
                    error = "Failed (${cov.code()})"
                }
                val fac = api.getOpsSupplyFactory()
                if (fac.isSuccessful) {
                    factory = fac.body()
                } else {
                    factoryError = "factory_unassigned (${fac.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Coverage and supply") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text("Pins and cities are set by the supplier. Nearest factory comes from the engine.", style = MaterialTheme.typography.bodySmall)
                ListItem(
                    headlineContent = { Text("Mode") },
                    supportingContent = { Text(coverageModeLabel(coverage?.mode)) },
                )
                if (!coverage?.countryCode.isNullOrBlank()) {
                    ListItem(
                        headlineContent = { Text("Country") },
                        supportingContent = { Text(coverage?.countryCode.orEmpty()) },
                    )
                }
                Text("Cities", style = MaterialTheme.typography.titleSmall)
                if (coverage?.cities.isNullOrEmpty()) {
                    Text("Closest same-country matching (no city cells).", color = MaterialTheme.colorScheme.onSurfaceVariant)
                } else {
                    coverage?.cities?.forEach { city ->
                        Text(city.name, modifier = Modifier.fillMaxWidth())
                    }
                }
                Text("Pins", style = MaterialTheme.typography.titleSmall)
                if (coverage?.pins.isNullOrEmpty()) {
                    Text("No supplier pins on this warehouse.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                } else {
                    coverage?.pins?.forEach { pin ->
                        ListItem(
                            headlineContent = { Text(pin.targetType) },
                            supportingContent = { Text(pin.targetId) },
                        )
                    }
                }
                Text("Nearest factory", style = MaterialTheme.typography.titleSmall)
                val factoryId = factory?.factoryId.orEmpty()
                if (factoryId.isNotBlank()) {
                    ListItem(
                        headlineContent = { Text(factoryId) },
                        supportingContent = { Text("Engine · ${factory?.countryCode.orEmpty()}") },
                    )
                } else {
                    Text(factoryError ?: "No same-country factory assigned.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}

private fun coverageModeLabel(mode: String?): String = when (mode?.uppercase()) {
    "PINNED" -> "Pinned"
    "CITY_CELLS" -> "City cells"
    else -> "Closest in country"
}
