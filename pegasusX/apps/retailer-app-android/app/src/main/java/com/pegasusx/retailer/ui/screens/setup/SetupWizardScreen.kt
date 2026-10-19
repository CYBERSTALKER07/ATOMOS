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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.R

@Composable
fun SetupWizardScreen(
    onSetupComplete: () -> Unit
) {
    var currentStep by rememberSaveable { mutableStateOf(0) }
    
    // Tax Info
    var taxId by rememberSaveable { mutableStateOf("") }
    var storeName by rememberSaveable { mutableStateOf("") }
    
    // Location Info
    var addressText by rememberSaveable { mutableStateOf("") }
    
    // Logistics Info
    var receivingWindowOpen by rememberSaveable { mutableStateOf("") }
    var receivingWindowClose by rememberSaveable { mutableStateOf("") }
    var selectedAccessType by rememberSaveable { mutableStateOf("") }
    var ceilingHeightText by rememberSaveable { mutableStateOf("") }

    val steps = listOf("Tax & Business", "Location", "Logistics")

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.White)
            .imePadding()
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
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
                steps.forEachIndexed { index, title ->
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
                modifier = Modifier.padding(top = 16.dp, bottom = 32.dp)
            )

            AnimatedContent(targetState = currentStep, label = "setup_step") { step ->
                Column(
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    when (step) {
                        0 -> {
                            OutlinedTextField(
                                value = storeName,
                                onValueChange = { storeName = it },
                                label = { Text("Store Name") },
                                modifier = Modifier.fillMaxWidth()
                            )
                            OutlinedTextField(
                                value = taxId,
                                onValueChange = { taxId = it },
                                label = { Text("Tax ID") },
                                modifier = Modifier.fillMaxWidth()
                            )
                        }
                        1 -> {
                            OutlinedTextField(
                                value = addressText,
                                onValueChange = { addressText = it },
                                label = { Text("Business Address") },
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

            Spacer(modifier = Modifier.weight(1f))
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                if (currentStep > 0) {
                    Button(onClick = { currentStep -= 1 }) {
                        Text("Back")
                    }
                } else {
                    Spacer(modifier = Modifier.weight(1f))
                }
                
                Button(
                    onClick = {
                        if (currentStep < steps.lastIndex) {
                            currentStep += 1
                        } else {
                            onSetupComplete()
                        }
                    }
                ) {
                    Text(if (currentStep < steps.lastIndex) "Next" else "Complete Setup")
                }
            }
        }
    }
}
