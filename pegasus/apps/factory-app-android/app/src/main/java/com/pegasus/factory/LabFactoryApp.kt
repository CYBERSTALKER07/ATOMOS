package com.pegasus.factory

import android.app.Application
import com.pegasus.factory.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class PegasusFactoryApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
    }
}
