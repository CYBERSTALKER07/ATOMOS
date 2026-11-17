package com.pegasusx.driver

import android.app.Application
import androidx.hilt.work.HiltWorkerFactory
import androidx.work.Configuration
import com.pegasusx.driver.data.remote.FirebaseAuthHelper
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.network.NetworkLifecycleEngine
import com.pegasusx.driver.util.DriverLocaleManager
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class PegasusDriverApp : Application(), Configuration.Provider {

    @Inject lateinit var workerFactory: HiltWorkerFactory

    override fun onCreate() {
        super.onCreate()
        DriverLocaleManager.applyProcessLocale(this)
        TokenHolder.init(this)
        FirebaseAuthHelper.init(this)
        NetworkLifecycleEngine.shared.initialize(this)
    }

    override val workManagerConfiguration: Configuration
        get() = Configuration.Builder()
            .setWorkerFactory(workerFactory)
            .build()
}
