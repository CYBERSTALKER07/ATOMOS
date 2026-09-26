import { useEffect } from 'react';
import { Alert } from 'react-native';
import * as Updates from 'expo-updates';

// ─── OTA Updates ──────────────────────────────────────────────────────────────

export function useOtaUpdates() {
  useEffect(() => {
    async function onFetchUpdateAsync() {
      try {
        const update = await Updates.checkForUpdateAsync();
        if (update.isAvailable) {
          await Updates.fetchUpdateAsync();
          Alert.alert(
            'Update Available',
            'A new version has been downloaded. Restart the app to apply the update?',
            [
              { text: 'Cancel', style: 'cancel' },
              { text: 'Restart', onPress: () => Updates.reloadAsync() }
            ]
          );
        }
      } catch (error) {
        // Silent fail on OTA check (network error, etc.)
        console.log(`Error fetching latest Expo update: ${error}`);
      }
    }
    if (!__DEV__) {
      onFetchUpdateAsync();
    }
  }, []);
}
