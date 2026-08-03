/**
 * Web Push notification initializer for registering browser FCM device tokens.
 * Interacts with backend Go POST /v1/user/device-token.
 */

export interface RegisterWebPushOptions {
  apiBaseUrl: string;
  authToken: string;
  vapidKey?: string;
}

export async function postDeviceToken(
  apiBaseUrl: string,
  authToken: string,
  token: string,
  platform: 'web' | 'desktop' = 'web',
): Promise<boolean> {
  const url = `${apiBaseUrl.replace(/\/$/, '')}/v1/user/device-token`;
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${authToken}`,
      },
      body: JSON.stringify({
        token: token.trim(),
        platform: platform.toUpperCase(),
      }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (typeof window === 'undefined' || !('Notification' in window)) {
    return 'denied';
  }
  if (Notification.permission === 'granted') {
    return 'granted';
  }
  return await Notification.requestPermission();
}
