import re

target1 = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/domain/api/PegasusApi.kt"
with open(target1, "r") as f:
    text1 = f.read()

target2 = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/domain/repository/PegasusApiRepo.kt"
with open(target2, "r") as f:
    text2 = f.read()

# Let's see if getSavedCards is present
print("PegasusApi.kt has getSavedCards:", "fun getSavedCards" in text1)
print("PegasusApiRepo.kt has getSavedCards:", "fun getSavedCards" in text2)
