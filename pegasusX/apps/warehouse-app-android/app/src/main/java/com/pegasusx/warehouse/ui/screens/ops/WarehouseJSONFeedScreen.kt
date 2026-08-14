package com.pegasusx.warehouse.ui.screens.ops

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import retrofit2.Response

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WarehouseJSONFeedScreen(
    title: String,
    onBack: (() -> Unit)?,
    loader: suspend () -> Response<JsonElement>,
) {
    var body by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = loader()
                if (resp.isSuccessful) {
                    body = resp.body()?.toString() ?: "{}"
                } else {
                    error = if (resp.code() == 503) "Unavailable" else "Failed (${resp.code()})"
                    body = null
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
                body = null
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(title) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title) },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = title,
                body = "Loading live API.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && body == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(body ?: "{}", modifier = Modifier.fillMaxWidth())
            }
        }
    }
}
