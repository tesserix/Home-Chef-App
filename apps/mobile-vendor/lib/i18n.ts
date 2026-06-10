import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import AsyncStorage from '@react-native-async-storage/async-storage';
import en from '../locales/en.json';
import hi from '../locales/hi.json';

export const SUPPORTED_LOCALES = ['en', 'hi'] as const;
export type AppLocale = (typeof SUPPORTED_LOCALES)[number];

const STORAGE_KEY = 'app.locale';

function isSupported(code: string | null | undefined): code is AppLocale {
  return !!code && (SUPPORTED_LOCALES as readonly string[]).includes(code);
}

/** Best language for the device, falling back to English. Defensive: a missing
 *  or misbehaving native locale module must never break i18n init — if locale
 *  detection throws, we just default to English. */
function deviceLocale(): AppLocale {
  try {
    // Lazy require — if the expo-localization NATIVE module is missing (e.g. a
    // JS-only update shipped before the binary was rebuilt), a top-level import
    // would throw at module load and redbox the whole app. require() inside the
    // try lets us catch that and fall back to English instead.
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const Localization = require('expo-localization');
    const code = Localization.getLocales()[0]?.languageCode;
    return isSupported(code) ? code : 'en';
  } catch {
    return 'en';
  }
}

// Initialise synchronously with the device locale so the first render is
// already translated; the persisted choice (if any) is applied on mount via
// hydratePersistedLocale().
i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    hi: { translation: hi },
  },
  lng: deviceLocale(),
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
  returnNull: false,
});

/** Apply the user's saved locale (overrides the device default) if present. */
export async function hydratePersistedLocale(): Promise<void> {
  try {
    const saved = await AsyncStorage.getItem(STORAGE_KEY);
    if (isSupported(saved) && i18n.language !== saved) {
      await i18n.changeLanguage(saved);
    }
  } catch {
    // best-effort — fall back to the device locale already applied
  }
}

/** The currently-selected locale, or 'system' when following the device. */
export async function getStoredLocalePreference(): Promise<AppLocale | 'system'> {
  try {
    const saved = await AsyncStorage.getItem(STORAGE_KEY);
    return isSupported(saved) ? saved : 'system';
  } catch {
    return 'system';
  }
}

/** Change the app language. 'system' clears the override and follows the device. */
export async function setLocale(locale: AppLocale | 'system'): Promise<void> {
  if (locale === 'system') {
    await AsyncStorage.removeItem(STORAGE_KEY);
    await i18n.changeLanguage(deviceLocale());
    return;
  }
  await AsyncStorage.setItem(STORAGE_KEY, locale);
  await i18n.changeLanguage(locale);
}

export default i18n;
