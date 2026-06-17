declare module '@tauri-apps/api/core';
declare module '@tauri-apps/api/event';
declare module 'firebase/app';
declare module 'firebase/auth' {
    export interface User {
        uid: string;
        getIdToken(forceRefresh?: boolean): Promise<string>;
        [key: string]: unknown;
    }

    export interface Auth {
        currentUser: User | null;
        signOut(): Promise<void>;
        [key: string]: unknown;
    }

    export interface UserCredential {
        user: User;
        [key: string]: unknown;
    }

    export interface ConfirmationResult {
        confirm(verificationCode: string): Promise<UserCredential>;
        [key: string]: unknown;
    }

    export function getAuth(app?: unknown): Auth;
    export function signInWithCustomToken(auth: Auth, customToken: string): Promise<UserCredential>;
    export function signInWithPhoneNumber(
        auth: Auth,
        phoneNumber: string,
        appVerifier: unknown
    ): Promise<ConfirmationResult>;
    export function signOut(auth: Auth): Promise<void>;
    export function onAuthStateChanged(auth: Auth, listener: (user: User | null) => void): () => void;
    export function connectAuthEmulator(auth: Auth, url: string, options?: { disableWarnings?: boolean }): void;
}
