package com.thelab.warehouse

import android.app.Application
import com.thelab.warehouse.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class LabWarehouseApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
    }
}
