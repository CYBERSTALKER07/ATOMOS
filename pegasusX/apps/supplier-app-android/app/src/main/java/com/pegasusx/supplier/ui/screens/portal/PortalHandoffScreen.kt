package com.pegasusx.supplier.ui.screens.portal

import androidx.compose.ui.res.stringResource

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Computer
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.ui.portal.SupplierPortalFeature
import com.pegasusx.supplier.ui.portal.SupplierPortalLinks
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PortalHandoffScreen(
    feature: SupplierPortalFeature,
    onBack: (() -> Unit)? = null,
) {
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(feature.titleRes)) },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = PegasusSpacing.xl),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(
                imageVector = Icons.Default.Computer,
                contentDescription = null,
                modifier = Modifier.size(56.dp),
                tint = MaterialTheme.colorScheme.primary,
            )
            Spacer(Modifier.height(PegasusSpacing.lg))
            Text(
                text = stringResource(R.string.mobile_supplier_ui_manage_on_web_portal),
                style = MaterialTheme.typography.titleLarge,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(PegasusSpacing.sm))
            Text(
                text = feature.handoffMessage,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(PegasusSpacing.xxl))
            Button(
                onClick = {
                    val intent = Intent(Intent.ACTION_VIEW, SupplierPortalLinks.openUri(feature))
                    context.startActivity(intent)
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("Open supplier portal")
            }
        }
    }
}
