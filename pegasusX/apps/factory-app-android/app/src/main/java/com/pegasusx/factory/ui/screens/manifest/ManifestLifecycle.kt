package com.pegasusx.factory.ui.screens.manifest

import com.pegasusx.factory.data.model.ManifestTransitionRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.util.FactoryIdempotencyKeys
import retrofit2.Response
import com.pegasusx.factory.data.model.ManifestTransitionResponse

data class ManifestLifecycleStep(
    val path: String,
    val label: String,
)

fun nextManifestLifecycleStep(state: String): ManifestLifecycleStep? = when (state.uppercase()) {
    "DRAFT" -> ManifestLifecycleStep("start-loading", "Start loading")
    "LOADING" -> ManifestLifecycleStep("seal", "Seal manifest")
    "SEALED" -> ManifestLifecycleStep("dispatch", "Dispatch")
    "DISPATCHED" -> ManifestLifecycleStep("complete", "Complete")
    else -> null
}

suspend fun applyManifestLifecycle(
    api: FactoryApi,
    manifestId: String,
    step: ManifestLifecycleStep,
): Response<ManifestTransitionResponse> {
    val body = ManifestTransitionRequest(reason = "factory-app-android")
    val idempotencyKey = FactoryIdempotencyKeys.forLifecyclePath(manifestId, step.path)
    return when (step.path) {
        "start-loading" -> api.startManifestLoading(manifestId, idempotencyKey, body)
        "seal" -> api.sealManifest(manifestId, idempotencyKey, body)
        "dispatch" -> api.dispatchManifest(manifestId, idempotencyKey, body)
        "complete" -> api.completeManifest(manifestId, idempotencyKey, body)
        else -> error("unsupported manifest lifecycle path: ${step.path}")
    }
}
