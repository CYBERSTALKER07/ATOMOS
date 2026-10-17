package com.pegasusx.factory

import android.app.Application
import com.pegasusx.factory.data.push.DeviceTokenRegistrar
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FirebaseAuthHelper
import com.pegasusx.factory.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class PegasusXFactoryApp : Application() {
    @Inject lateinit var api: FactoryApi

    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
        FirebaseAuthHelper.init(this)
        DeviceTokenRegistrar.uploadBestEffort(api)
    }
}
