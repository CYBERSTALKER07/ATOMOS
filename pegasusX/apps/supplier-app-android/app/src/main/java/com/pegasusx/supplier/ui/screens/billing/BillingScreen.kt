package com.pegasusx.supplier.ui.screens.billing

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.BillingSetupRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillingScreen(
    api: SupplierApi,
    onSkip: () -> Unit,
    onComplete: () -> Unit,
) {
    var bankName by remember { mutableStateOf("") }
    var accountHolder by remember { mutableStateOf("") }
    var accountNumber by remember { mutableStateOf("") }
    var swiftBic by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = { TopAppBar(title = { Text("Billing setup") }) },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Configure banking and payment gateway (or skip for now).",
                style = MaterialTheme.typography.bodyMedium,
            )
            OutlinedTextField(bankName, { bankName = it }, label = { Text("Bank name") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(accountHolder, { accountHolder = it }, label = { Text("Account holder") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(accountNumber, { accountNumber = it }, label = { Text("Account number") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(swiftBic, { swiftBic = it }, label = { Text("SWIFT/BIC") }, modifier = Modifier.fillMaxWidth())
            if (error != null) {
                Text(error!!, color = MaterialTheme.colorScheme.error)
            }
            Button(
                onClick = {
                    loading = true
                    error = null
                    scope.launch {
                        try {
                            val resp = api.configureBilling(
                                BillingSetupRequest(
                                    bankName = bankName,
                                    accountHolder = accountHolder,
                                    accountNumber = accountNumber,
                                    swiftBic = swiftBic,
                                    iban = null,
                                    selectedGateways = listOf("payme"),
                                ),
                            )
                            if (resp.isSuccessful) onComplete()
                            else error = "Setup failed (${resp.code()})"
                        } catch (e: Exception) {
                            error = e.message
                        } finally {
                            loading = false
                        }
                    }
                },
                enabled = !loading,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Save billing") }
            OutlinedButton(onClick = onSkip, modifier = Modifier.fillMaxWidth()) {
                Text("Skip for now")
            }
        }
    }
}
