package com.pegasus.warehouse

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.navigation.WarehouseNavigation
import com.pegasus.warehouse.ui.theme.LabWarehouseTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var warehouseApi: WarehouseApi

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            LabWarehouseTheme {
                WarehouseNavigation(api = warehouseApi)
            }
        }
    }
}
