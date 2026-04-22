# ProGuard/R8 rules for LabPayload release builds.
-keepattributes Signature, *Annotation*, EnclosingMethod, InnerClasses
-keep class kotlinx.serialization.** { *; }
-keep class com.thelab.payload.data.model.** { *; }
-dontwarn kotlinx.serialization.**
-dontwarn org.bouncycastle.**
-dontwarn org.conscrypt.**
-dontwarn org.openjsse.**
