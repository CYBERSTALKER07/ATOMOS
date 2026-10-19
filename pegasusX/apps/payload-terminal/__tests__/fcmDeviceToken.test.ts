import { describe, it, expect } from 'vitest';
import { fcmRegistrationFromNativePush } from '../fcmDeviceToken';

describe('fcmRegistrationFromNativePush', () => {
  it('accepts Android FCM registration tokens', () => {
    expect(
      fcmRegistrationFromNativePush({ type: 'android', data: 'fcm-reg-abc' }),
    ).toEqual({ token: 'fcm-reg-abc', platform: 'android' });
  });

  it('rejects Expo push tokens', () => {
    expect(
      fcmRegistrationFromNativePush({
        type: 'android',
        data: 'ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]',
      }),
    ).toBeNull();
    expect(
      fcmRegistrationFromNativePush({
        type: 'expo',
        data: 'ExpoPushToken[yyyy]',
      }),
    ).toBeNull();
  });

  it('rejects iOS APNs device tokens (not FCM)', () => {
    expect(
      fcmRegistrationFromNativePush({ type: 'ios', data: 'abcdef0123456789' }),
    ).toBeNull();
  });

  it('rejects empty or missing tokens', () => {
    expect(fcmRegistrationFromNativePush(null)).toBeNull();
    expect(fcmRegistrationFromNativePush({ type: 'android', data: '  ' })).toBeNull();
  });
});
