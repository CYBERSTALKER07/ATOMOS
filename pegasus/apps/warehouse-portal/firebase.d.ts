declare module '@tauri-apps/api/core';
declare module '@tauri-apps/api/event';
declare module 'firebase/app';
declare module 'firebase/auth' {
    export type User = any;
    export type Auth = any;
    export const getAuth: any;
    export const signInWithCustomToken: any;
    export const signOut: any;
    export const onAuthStateChanged: any;
    export const connectAuthEmulator: any;
}
