package com.pegasusx.retailer

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.material3.windowsizeclass.ExperimentalMaterial3WindowSizeClassApi
import androidx.compose.material3.windowsizeclass.calculateWindowSizeClass
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.ui.navigation.RetailerNavigation
import com.pegasusx.retailer.ui.screens.auth.AuthScreen
import com.pegasusx.retailer.ui.theme.PegasusRetailerTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var tokenManager: TokenManager

    @OptIn(ExperimentalMaterial3WindowSizeClassApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val windowSizeClass = calculateWindowSizeClass(this)
            PegasusRetailerTheme {
                var isAuthenticated by rememberSaveable {
                    mutableStateOf(tokenManager.getToken() != null)
                }

                if (isAuthenticated) {
                    RetailerNavigation(windowSizeClass = windowSizeClass)
                } else {
                    AuthScreen(onAuthenticated = { isAuthenticated = true })
                }
            }
        }
    }
}
