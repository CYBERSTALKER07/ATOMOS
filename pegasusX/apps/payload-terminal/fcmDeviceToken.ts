/** Native device push token from expo-notifications getDevicePushTokenAsync. */
export type NativePushToken = {
  type?: string;
  data?: string;
};

/**
 * FCM Admin Send only accepts FCM registration tokens.
 * Expo push tokens (ExponentPushToken[…]) and iOS APNs device tokens are not FCM.
 * Android getDevicePushTokenAsync data is an FCM registration token.
 */
export function fcmRegistrationFromNativePush(
  token: NativePushToken | null | undefined,
): { token: string; platform: 'android' } | null {
  if (!token) return null;
  const type = String(token.type ?? '').toLowerCase();
  const data = String(token.data ?? '').trim();
  if (!data) return null;
  if (data.startsWith('ExponentPushToken[') || data.startsWith('ExpoPushToken[')) {
    return null;
  }
  if (type === 'android') {
    return { token: data, platform: 'android' };
  }
  return null;
}
