package com.pegasusx.factory

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.material3.windowsizeclass.ExperimentalMaterial3WindowSizeClassApi
import androidx.compose.material3.windowsizeclass.calculateWindowSizeClass
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.GeocodeApi
import com.pegasusx.factory.ui.navigation.FactoryNavigation
import com.pegasusx.factory.ui.theme.PegasusFactoryTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var factoryApi: FactoryApi
    @Inject lateinit var geocodeApi: GeocodeApi

    @OptIn(ExperimentalMaterial3WindowSizeClassApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val windowSizeClass = calculateWindowSizeClass(this)
            PegasusFactoryTheme {
                FactoryNavigation(api = factoryApi, geocodeApi = geocodeApi, windowSizeClass = windowSizeClass)
            }
        }
    }
}
