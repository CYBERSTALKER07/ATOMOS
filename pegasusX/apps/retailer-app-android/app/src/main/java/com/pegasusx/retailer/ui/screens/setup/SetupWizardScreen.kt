package com.pegasusx.retailer.ui.screens.setup

import androidx.compose.ui.res.stringResource

import androidx.compose.animation.AnimatedContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.retailer.R
import kotlinx.coroutines.launch

@Composable
fun SetupWizardScreen(
    onSetupComplete: () -> Unit,
    viewModel: SetupWizardViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    var currentStep by rememberSaveable { mutableStateOf(0) }

    // Logistics Info (preserved UI step)
    var receivingWindowOpen by rememberSaveable { mutableStateOf("") }
    var receivingWindowClose by rememberSaveable { mutableStateOf("") }
    var selectedAccessType by rememberSaveable { mutableStateOf("") }
    var ceilingHeightText by rememberSaveable { mutableStateOf("") }

    LaunchedEffect(uiState.setupCompleted) {
        if (uiState.setupCompleted) {
            onSetupComplete()
        }
    }

    val steps = listOf("Tax & Business", "Address", "Logistics")

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.White)
            .imePadding()
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(modifier = Modifier.height(32.dp))

            Text(
                text = stringResource(R.string.mobile_retailer_ui_retailer_setup),
                style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.onSurface
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Progress Indicator
            Row(
                horizontalArrangement = Arrangement.Center,
                modifier = Modifier.fillMaxWidth()
            ) {
                steps.forEachIndexed { index, _ ->
                    val color = if (index <= currentStep) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant
                    Box(
                        modifier = Modifier
                            .padding(horizontal = 4.dp)
                            .height(8.dp)
                            .weight(1f)
                            .clip(CircleShape)
                            .background(color)
                    )
                }
            }

            Text(
                text = steps[currentStep],
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 16.dp, bottom = 24.dp)
            )

            if (uiState.joinRequestSent) {
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Text(
                            text = stringResource(R.string.supplier_portal_onboarding_inn_existing_company_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            text = stringResource(R.string.supplier_portal_onboarding_inn_join_request_sent),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        uiState.joinRequestConflict?.legalName?.takeIf { it.isNotBlank() }?.let { legal ->
                            Text(
                                text = legal,
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                        uiState.joinRequestConflict?.requestId?.takeIf { it.isNotBlank() }?.let { reqId ->
                            Text(
                                text = "Request ID: $reqId",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        OutlinedButton(
                            onClick = {
                                viewModel.clearJoinRequestState()
                                currentStep = 0
                            },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text("Back")
                        }
                    }
                }
            } else {
                AnimatedContent(targetState = currentStep, label = "setup_step") { step ->
                    Column(
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        when (step) {
                            0 -> {
                                Text(
                                    text = stringResource(R.string.retailer_desktop_tax_identity_subtitle),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Card(modifier = Modifier.fillMaxWidth()) {
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(14.dp),
                                        verticalAlignment = Alignment.CenterVertically,
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Column(modifier = Modifier.weight(1f).padding(end = 12.dp)) {
                                            Text(
                                                text = stringResource(R.string.retailer_desktop_tax_identity_mode_informal_phone),
                                                style = MaterialTheme.typography.titleSmall,
                                                fontWeight = FontWeight.SemiBold,
                                            )
                                            Text(
                                                text = stringResource(R.string.retailer_desktop_tax_identity_legal_status_informal),
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            )
                                        }
                                        Switch(
                                            checked = uiState.isInformalShop,
                                            onCheckedChange = { viewModel.setInformalShop(it) },
                                        )
                                    }
                                }

                                if (!uiState.isInformalShop) {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        OutlinedTextField(
                                            value = uiState.taxId,
                                            onValueChange = { viewModel.setTaxId(it) },
                                            label = { Text(stringResource(R.string.retailer_desktop_tax_identity_tax_id_label)) },
                                            placeholder = { Text(stringResource(R.string.supplier_portal_onboarding_inn_tax_id_placeholder)) },
                                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                                            singleLine = true,
                                            modifier = Modifier.weight(1f),
                                        )
                                        OutlinedButton(
                                            onClick = { scope.launch { viewModel.verifyInn() } },
                                            enabled = !uiState.innChecking && uiState.taxId.isNotBlank(),
                                        ) {
                                            if (uiState.innChecking) {
                                                CircularProgressIndicator(modifier = Modifier.height(18.dp))
                                            } else {
                                                Text(stringResource(R.string.supplier_portal_onboarding_inn_verify_button))
                                            }
                                        }
                                    }

                                    uiState.innLookup?.takeIf { it.isVerifiedFound }?.let { lookup ->
                                        Card(
                                            colors = CardDefaults.cardColors(
                                                containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f),
                                            ),
                                            modifier = Modifier.fillMaxWidth(),
                                        ) {
                                            Column(
                                                modifier = Modifier.padding(12.dp),
                                                verticalArrangement = Arrangement.spacedBy(4.dp),
                                            ) {
                                                Text(
                                                    text = stringResource(R.string.supplier_portal_onboarding_inn_verified_badge),
                                                    style = MaterialTheme.typography.labelMedium,
                                                    fontWeight = FontWeight.Bold,
                                                    color = MaterialTheme.colorScheme.primary,
                                                )
                                                lookup.legalName?.takeIf { it.isNotBlank() }?.let { legal ->
                                                    Text(legal, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
                                                }
                                                lookup.address?.takeIf { it.isNotBlank() }?.let { addr ->
                                                    Text(addr, style = MaterialTheme.typography.bodySmall)
                                                }
                                                if (lookup.existingJoinEligible == true) {
                                                    Text(
                                                        text = stringResource(R.string.retailer_desktop_tax_identity_existing_retailer_join_hint),
                                                        style = MaterialTheme.typography.bodySmall,
                                                        color = MaterialTheme.colorScheme.error,
                                                    )
                                                }
                                            }
                                        }
                                    }

                                    uiState.innError?.let { err ->
                                        Text(
                                            text = err,
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.error,
                                        )
                                    }
                                }
                            }
                            1 -> {
                                OutlinedTextField(
                                    value = uiState.billingAddress,
                                    onValueChange = { viewModel.setBillingAddress(it) },
                                    label = { Text("Billing Address") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                                OutlinedTextField(
                                    value = uiState.shippingAddress,
                                    onValueChange = { viewModel.setShippingAddress(it) },
                                    label = { Text("Shipping Address") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                                OutlinedTextField(
                                    value = uiState.city,
                                    onValueChange = { viewModel.setCity(it) },
                                    label = { Text("City") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                                OutlinedTextField(
                                    value = uiState.postalCode,
                                    onValueChange = { viewModel.setPostalCode(it) },
                                    label = { Text("Postal Code") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                            }
                            2 -> {
                                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    OutlinedTextField(
                                        value = receivingWindowOpen,
                                        onValueChange = { receivingWindowOpen = it },
                                        label = { Text("Open (e.g. 09:00)") },
                                        modifier = Modifier.weight(1f)
                                    )
                                    OutlinedTextField(
                                        value = receivingWindowClose,
                                        onValueChange = { receivingWindowClose = it },
                                        label = { Text("Close (e.g. 18:00)") },
                                        modifier = Modifier.weight(1f)
                                    )
                                }
                                OutlinedTextField(
                                    value = selectedAccessType,
                                    onValueChange = { selectedAccessType = it },
                                    label = { Text("Access Type (e.g. Loading Dock)") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                                OutlinedTextField(
                                    value = ceilingHeightText,
                                    onValueChange = { ceilingHeightText = it },
                                    label = { Text("Ceiling Height (m)") },
                                    modifier = Modifier.fillMaxWidth()
                                )
                            }
                        }
                    }
                }

                uiState.submitError?.let { err ->
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(
                        text = err,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                Spacer(modifier = Modifier.weight(1f))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    if (currentStep > 0) {
                        Button(
                            onClick = { currentStep -= 1 },
                            enabled = !uiState.isSubmitting,
                        ) {
                            Text("Back")
                        }
                    } else {
                        Spacer(modifier = Modifier.weight(1f))
                    }

                    Button(
                        enabled = !uiState.innChecking && !uiState.isSubmitting,
                        onClick = {
                            scope.launch {
                                when (currentStep) {
                                    0 -> if (viewModel.canAdvanceFromTax()) currentStep += 1
                                    1 -> currentStep += 1
                                    else -> viewModel.submitSetup()
                                }
                            }
                        }
                    ) {
                        if (uiState.isSubmitting) {
                            CircularProgressIndicator(modifier = Modifier.height(18.dp), color = Color.White)
                        } else if (currentStep == 0 && uiState.isInformalShop) {
                            Text(stringResource(R.string.retailer_desktop_tax_identity_save_informal))
                        } else if (currentStep < steps.lastIndex) {
                            Text("Next")
                        } else {
                            Text(stringResource(R.string.mobile_retailer_ui_complete_setup))
                        }
                    }
                }
            }
        }
    }
}
