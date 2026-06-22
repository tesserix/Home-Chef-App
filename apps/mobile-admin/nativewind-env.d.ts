/// <reference types="nativewind/types" />

// SDK 56 (NativeWind 4.2 / RN 0.86): the bundled types no longer declare a
// side-effect module for `*.css`, so `import '../global.css'` reports TS2882.
// Declare the CSS side-effect module so the global stylesheet import resolves.
declare module '*.css' {}
