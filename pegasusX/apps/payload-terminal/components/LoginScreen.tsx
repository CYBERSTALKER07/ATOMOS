import type { ReactNode } from 'react';
import { Text, TextInput, View } from 'react-native';

import { isIOS, type AppTheme } from '../theme';
import Pressable from './Pressable';

// ─── Render: PAYLOADER LOGIN ──────────────────────────────────────────────────

export default function LoginScreen({
  theme: T,
  tx,
  loginMode,
  phoneInput,
  onPhoneInputChange,
  pinInput,
  onPinInputChange,
  otpInput,
  onOtpInputChange,
  otpSent,
  isLoggingIn,
  onSendOtp,
  onVerifyOtp,
  onLoginPin,
  onToggleLoginMode,
  policyBanner,
  toast,
}: {
  theme: AppTheme;
  tx: (key: string) => string;
  loginMode: 'otp' | 'pin';
  phoneInput: string;
  onPhoneInputChange: (value: string) => void;
  pinInput: string;
  onPinInputChange: (value: string) => void;
  otpInput: string;
  onOtpInputChange: (value: string) => void;
  otpSent: boolean;
  isLoggingIn: boolean;
  onSendOtp: () => void;
  onVerifyOtp: () => void;
  onLoginPin: () => void;
  onToggleLoginMode: () => void;
  policyBanner: ReactNode;
  toast: ReactNode;
}) {
  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background }}>
      {policyBanner}
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 48 }}>
      <Text style={{ fontWeight: '700', fontSize: 14, color: T.colors.tertiaryLabel, letterSpacing: 0.5, marginBottom: 32 }}>
        {tx('auth.login.payload_terminal')}
      </Text>
      <Text style={{ fontSize: 22, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0.5, marginBottom: 32 }}>
        {tx('auth.login.payloader_login')}
      </Text>
      <Text style={{ fontSize: 13, color: T.colors.secondaryLabel, marginBottom: 16, textAlign: 'center' }}>
        {loginMode === 'otp'
          ? (isIOS ? 'Sign in with warehouse phone OTP.' : 'SIGN IN WITH WAREHOUSE PHONE OTP.')
          : (isIOS ? 'Dev login with phone and PIN.' : 'DEV LOGIN WITH PHONE AND PIN.')}
      </Text>
      <TextInput
        placeholder={tx('common.field.phone')}
        placeholderTextColor={T.colors.tertiaryLabel}
        value={phoneInput}
        onChangeText={onPhoneInputChange}
        keyboardType="phone-pad"
        autoCapitalize="none"
        editable={!isLoggingIn && (loginMode === 'pin' || !otpSent)}
        style={{
          width: 320,
          borderWidth: isIOS ? 0.33 : 1,
          borderColor: T.colors.separator,
          backgroundColor: T.colors.cardBackground,
          borderRadius: T.radius.card,
          paddingHorizontal: 16,
          paddingVertical: 14,
          fontSize: 15,
          color: T.colors.label,
          marginBottom: 12,
        }}
      />
      {loginMode === 'otp' && otpSent ? (
        <TextInput
          placeholder={isIOS ? 'Verification code' : 'VERIFICATION CODE'}
          placeholderTextColor={T.colors.tertiaryLabel}
          value={otpInput}
          onChangeText={onOtpInputChange}
          keyboardType="number-pad"
          maxLength={6}
          style={{
            width: 320,
            borderWidth: isIOS ? 0.33 : 1,
            borderColor: T.colors.separator,
            backgroundColor: T.colors.cardBackground,
            borderRadius: T.radius.card,
            paddingHorizontal: 16,
            paddingVertical: 14,
            fontSize: 15,
            color: T.colors.label,
            marginBottom: 24,
            letterSpacing: 8,
            textAlign: 'center',
          }}
        />
      ) : null}
      {loginMode === 'pin' ? (
      <TextInput
        placeholder={tx('auth.login.pin_label')}
        placeholderTextColor={T.colors.tertiaryLabel}
        value={pinInput}
        onChangeText={onPinInputChange}
        keyboardType="number-pad"
        maxLength={8}
        secureTextEntry
        style={{
          width: 320,
          borderWidth: isIOS ? 0.33 : 1,
          borderColor: T.colors.separator,
          backgroundColor: T.colors.cardBackground,
          borderRadius: T.radius.card,
          paddingHorizontal: 16,
          paddingVertical: 14,
          fontSize: 15,
          color: T.colors.label,
          marginBottom: 24,
          letterSpacing: 8,
          textAlign: 'center',
        }}
      />
      ) : null}
      {loginMode === 'otp' ? (
        !otpSent ? (
          <Pressable
            onPress={onSendOtp}
            disabled={isLoggingIn || !phoneInput.trim()}
            style={({ pressed }) => ({
              width: 320,
              paddingVertical: 16,
              alignItems: 'center' as const,
              backgroundColor: !isLoggingIn && phoneInput.trim() ? T.colors.accent : T.colors.fillSecondary,
              borderRadius: T.radius.button,
              opacity: pressed ? 0.82 : 1,
              transform: [{ scale: pressed ? 0.97 : 1 }],
              marginBottom: 12,
            })}
          >
            <Text style={{
              fontWeight: '600',
              fontSize: 14,
              letterSpacing: isIOS ? 0.3 : 1,
              color: !isLoggingIn && phoneInput.trim() ? '#FFFFFF' : T.colors.tertiaryLabel,
            }}>
              {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Send Code' : 'SEND CODE')}
            </Text>
          </Pressable>
        ) : (
          <>
            <Pressable
              onPress={onVerifyOtp}
              disabled={isLoggingIn || otpInput.trim().length < 6}
              style={({ pressed }) => ({
                width: 320,
                paddingVertical: 16,
                alignItems: 'center' as const,
                backgroundColor: !isLoggingIn && otpInput.trim().length >= 6 ? T.colors.accent : T.colors.fillSecondary,
                borderRadius: T.radius.button,
                opacity: pressed ? 0.82 : 1,
                transform: [{ scale: pressed ? 0.97 : 1 }],
                marginBottom: 12,
              })}
            >
              <Text style={{
                fontWeight: '600',
                fontSize: 14,
                letterSpacing: isIOS ? 0.3 : 1,
                color: !isLoggingIn && otpInput.trim().length >= 6 ? '#FFFFFF' : T.colors.tertiaryLabel,
              }}>
                {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Verify & Sign In' : 'VERIFY & SIGN IN')}
              </Text>
            </Pressable>
            <Pressable onPress={onSendOtp} disabled={isLoggingIn} style={{ marginBottom: 12 }}>
              <Text style={{ color: T.colors.secondaryLabel, fontSize: 13 }}>
                {isIOS ? 'Resend code' : 'RESEND CODE'}
              </Text>
            </Pressable>
          </>
        )
      ) : (
      <Pressable
        onPress={onLoginPin}
        disabled={isLoggingIn || !phoneInput || pinInput.length < 6}
        style={({ pressed }) => ({
          width: 320,
          paddingVertical: 16,
          alignItems: 'center' as const,
          backgroundColor: !isLoggingIn && phoneInput && pinInput.length >= 6 ? T.colors.accent : T.colors.fillSecondary,
          borderRadius: T.radius.button,
          opacity: pressed ? 0.82 : 1,
          transform: [{ scale: pressed ? 0.97 : 1 }],
          marginBottom: 12,
        })}
      >
        <Text style={{
          fontWeight: '600',
          fontSize: 14,
          letterSpacing: isIOS ? 0.3 : 1,
          color: !isLoggingIn && phoneInput && pinInput.length >= 6 ? '#FFFFFF' : T.colors.tertiaryLabel,
        }}>
          {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Sign In with PIN' : 'SIGN IN WITH PIN')}
        </Text>
      </Pressable>
      )}
      <Pressable
        onPress={onToggleLoginMode}
        disabled={isLoggingIn}
      >
        <Text style={{ color: T.colors.secondaryLabel, fontSize: 13 }}>
          {loginMode === 'otp' ? (isIOS ? 'Use PIN (dev)' : 'USE PIN (DEV)') : (isIOS ? 'Use phone OTP' : 'USE PHONE OTP')}
        </Text>
      </Pressable>
      {toast}
      </View>
    </View>
  );
}
