import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import { authFetch } from './authSession';
import { fcmRegistrationFromNativePush } from './fcmDeviceToken';

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

/** Register an FCM registration token. Expo push tokens are not FCM — do not POST them. */
export async function registerPayloadPushTokens(): Promise<void> {
  if (!Device.isDevice) return;

  const { status: existing } = await Notifications.getPermissionsAsync();
  let finalStatus = existing;
  if (existing !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }
  if (finalStatus !== 'granted') return;

  try {
    const native = await Notifications.getDevicePushTokenAsync();
    const fcm = fcmRegistrationFromNativePush(native);
    if (!fcm) return;
    await postDeviceToken(fcm.token, fcm.platform);
  } catch {
    // Simulator / iOS APNs / missing google-services: no FCM token to store.
  }
}
