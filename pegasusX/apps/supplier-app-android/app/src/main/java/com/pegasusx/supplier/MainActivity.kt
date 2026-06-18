package com.pegasusx.supplier

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.compose.material3.windowsizeclass.ExperimentalMaterial3WindowSizeClassApi
import androidx.compose.material3.windowsizeclass.calculateWindowSizeClass
import androidx.lifecycle.lifecycleScope
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.SupplierWebSocket
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.data.remote.reconcileSupplierSession
import com.pegasusx.supplier.ui.navigation.SupplierNavigation
import com.pegasusx.supplier.ui.theme.PegasusSupplierTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject
import kotlinx.coroutines.launch
import org.maplibre.android.MapLibre

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var supplierApi: SupplierApi
    @Inject lateinit var supplierOps: SupplierOperationsRepository
    @Inject lateinit var geocodeApi: GeocodeApi
    @Inject lateinit var supplierWebSocket: SupplierWebSocket
    @Inject lateinit var realtimeSignals: SupplierRealtimeSignals

    @OptIn(ExperimentalMaterial3WindowSizeClassApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        MapLibre.getInstance(this)
        enableEdgeToEdge()
        if (TokenHolder.isLoggedIn) {
            supplierWebSocket.connect(BuildConfig.API_BASE_URL.trimEnd('/'))
        }
        lifecycleScope.launch {
            supplierWebSocket.messages.collect { msg ->
                if (!msg.type.startsWith("SYSTEM")) {
                    realtimeSignals.bump()
                }
            }
        }
        lifecycleScope.launch {
            supplierWebSocket.reconnects.collect {
                reconcileSupplierSession(supplierApi)
                realtimeSignals.bumpReconnect()
            }
        }
        setContent {
            val windowSizeClass = calculateWindowSizeClass(this)
            PegasusSupplierTheme {
                SupplierNavigation(
                    api = supplierApi,
                    ops = supplierOps,
                    geocodeApi = geocodeApi,
                    realtimeSignals = realtimeSignals,
                    windowSizeClass = windowSizeClass,
                )
            }
        }
    }
}
