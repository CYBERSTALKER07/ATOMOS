package com.pegasus.factory

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.ui.navigation.FactoryNavigation
import com.pegasus.factory.ui.theme.PegasusFactoryTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var factoryApi: FactoryApi

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            PegasusFactoryTheme {
                FactoryNavigation(api = factoryApi)
            }
        }
    }
}
