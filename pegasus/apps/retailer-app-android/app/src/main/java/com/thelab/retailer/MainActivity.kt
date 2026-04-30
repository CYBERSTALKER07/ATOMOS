package com.thelab.retailer

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
import com.thelab.retailer.data.local.TokenManager
import com.thelab.retailer.ui.navigation.RetailerNavigation
import com.thelab.retailer.ui.screens.auth.AuthScreen
import com.thelab.retailer.ui.theme.LabRetailerTheme
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
            LabRetailerTheme {
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
