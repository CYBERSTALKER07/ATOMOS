package com.pegasus.warehouse

import android.app.Application
import com.pegasus.warehouse.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class LabWarehouseApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
    }
}
