import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import Constants from 'expo-constants';
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import { authFetch } from './authSession';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

async function postDeviceToken(token: string, platform: string): Promise<void> {
  if (!token.trim()) return;
  try {
    await authFetch('/v1/user/device-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token, platform }),
    });
  } catch {
    // Push registration is best-effort; WS inbox remains primary.
  }
}

/** Register Expo push + any firebase_token returned at login (server-side FCM routing). */
export async function registerPayloadPushTokens(): Promise<void> {
  const firebaseToken = await SecureStore.getItemAsync('payloader_firebase_token');
  if (firebaseToken) {
    await postDeviceToken(firebaseToken, Platform.OS === 'ios' ? 'IOS' : 'ANDROID');
  }

  if (!Device.isDevice) return;

  const { status: existing } = await Notifications.getPermissionsAsync();
  let finalStatus = existing;
  if (existing !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }
  if (finalStatus !== 'granted') return;

  const projectId =
    Constants.expoConfig?.extra?.eas?.projectId ??
    Constants.easConfig?.projectId;
  const pushToken = await Notifications.getExpoPushTokenAsync(
    projectId ? { projectId } : undefined,
  );
  await postDeviceToken(pushToken.data, 'EXPO');
}
