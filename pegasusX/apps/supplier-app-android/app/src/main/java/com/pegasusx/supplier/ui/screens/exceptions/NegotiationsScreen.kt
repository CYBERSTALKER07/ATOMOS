package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.foundation.layout.padding
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane

/**
 * Quantity negotiation is product-disabled ecosystem-wide.
 * Kept as a dead-end empty state so accidental deep-links do not call resolve APIs.
 * Independent of claims / shop-closed / missing-items.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NegotiationsScreen(onBack: () -> Unit) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Negotiations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "Negotiations disabled",
            body = "Quantity negotiation is not available. Use shop-closed, claims, or missing-items for delivery exceptions.",
            modifier = Modifier.padding(padding),
        )
    }
}
