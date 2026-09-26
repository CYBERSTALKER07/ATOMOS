package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

data class ShiftRow(
    val shiftId: String,
    val status: String,
    val openingFloatMinor: Long,
    val varianceMinor: Long?,
)

@HiltViewModel
class ShiftsViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ShiftsScreen(
    onNavigateBack: () -> Unit,
    viewModel: ShiftsViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var clockedIn by remember { mutableStateOf(false) }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var floatMinor by remember { mutableStateOf("0") }
    var closingCash by remember { mutableStateOf("0") }
    var registerId by remember { mutableStateOf<String?>(null) }
    val shifts = remember { mutableStateListOf<ShiftRow>() }

    fun refresh() {
        scope.launch {
            try {
                val time = viewModel.api.getTimeEntries().asJsonObject
                clockedIn = time.get("clocked_in")?.asBoolean == true
                val shiftJson = viewModel.api.getShifts().asJsonObject
                val items = shiftJson.getAsJsonArray("items")
                shifts.clear()
                if (items != null) {
                    for (el in items) {
                        val o = el.asJsonObject
                        shifts.add(
                            ShiftRow(
                                shiftId = o.get("shift_id")?.asString ?: continue,
                                status = o.get("status")?.asString ?: "",
                                openingFloatMinor = o.get("opening_float_minor")?.asLong ?: 0L,
                                varianceMinor = o.get("variance_minor")?.takeIf { !it.isJsonNull }?.asLong,
                            ),
                        )
                    }
                }
                if (registerId == null) {
                    val regs = viewModel.api.getRegisters().asJsonObject.getAsJsonArray("items")
                    if (regs != null && regs.size > 0) {
                        registerId = regs[0].asJsonObject.get("register_id")?.asString
                    }
                }
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    LaunchedEffect(Unit) { refresh() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Shifts & time") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    "Clock in before opening POS when SHIFTS pack is on. Close shift with counted cash for variance alerts.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            if (clockedIn) "Clocked in" else "Not clocked in",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        if (!clockedIn) {
                            Button(enabled = !busy, onClick = {
                                scope.launch {
                                    busy = true
                                    try {
                                        viewModel.api.clockIn(body = emptyMap())
                                        banner = "Clocked in"
                                        refresh()
                                    } catch (e: Exception) {
                                        banner = e.message
                                    } finally {
                                        busy = false
                                    }
                                }
                            }) { Text("Clock in") }
                        } else {
                            OutlinedButton(enabled = !busy, onClick = {
                                scope.launch {
                                    busy = true
                                    try {
                                        viewModel.api.clockOut()
                                        banner = "Clocked out"
                                        refresh()
                                    } catch (e: Exception) {
                                        banner = e.message
                                    } finally {
                                        busy = false
                                    }
                                }
                            }) { Text("Clock out") }
                        }
                    }
                }
            }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Cash shift", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(
                            value = floatMinor,
                            onValueChange = { floatMinor = it },
                            label = { Text("Opening float (minor)") },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        OutlinedTextField(
                            value = closingCash,
                            onValueChange = { closingCash = it },
                            label = { Text("Closing cash (minor)") },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Button(enabled = !busy && clockedIn, onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    val body = mutableMapOf<String, Any>(
                                        "opening_float_minor" to (floatMinor.toLongOrNull() ?: 0L),
                                        "currency" to com.pegasusx.retailer.data.model.sessionPackCurrency(),
                                    )
                                    registerId?.let { body["register_id"] = it }
                                    viewModel.api.openShift(
                                        body = body,
                                        idempotencyKey = "shift-${System.currentTimeMillis()}",
                                    )
                                    banner = "Shift opened"
                                    refresh()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Open shift") }
                    }
                }
            }
            items(shifts) { row ->
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(stringResource(R.string.mobile_retailer_ui_status_float_n_0, row.status, row.openingFloatMinor / 100.0))
                        row.varianceMinor?.let {
                            Text(stringResource(R.string.mobile_retailer_ui_variance_n_0, it / 100.0), style = MaterialTheme.typography.bodySmall)
                        }
                        if (row.status == "OPEN") {
                            OutlinedButton(enabled = !busy, onClick = {
                                scope.launch {
                                    busy = true
                                    try {
                                        val closed = viewModel.api.closeShift(
                                            shiftId = row.shiftId,
                                            body = mapOf(
                                                "closing_cash_minor" to (closingCash.toLongOrNull() ?: 0L),
                                            ),
                                        ).asJsonObject
                                        val v = closed.get("variance_minor")?.takeIf { !it.isJsonNull }?.asLong
                                        banner = "Shift closed" + if (v != null) " · variance ${v / 100.0}" else ""
                                        refresh()
                                    } catch (e: Exception) {
                                        banner = e.message
                                    } finally {
                                        busy = false
                                    }
                                }
                            }) { Text("Close shift") }
                        }
                    }
                }
            }
        }
    }
}
