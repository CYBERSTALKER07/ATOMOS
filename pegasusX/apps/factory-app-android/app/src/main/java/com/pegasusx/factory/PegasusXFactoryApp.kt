package com.pegasusx.factory

import android.app.Application
import com.pegasusx.factory.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class PegasusXFactoryApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
    }
}
