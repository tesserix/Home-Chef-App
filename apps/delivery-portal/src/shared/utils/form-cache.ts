const CACHE_KEY = 'hc-driver-onboarding-draft';
const CACHE_TTL_MS = 48 * 60 * 60 * 1000; // 48 hours

interface CacheEntry {
  data: Record<string, Record<string, string>>;
  savedAt: number;
}

function getEntry(): CacheEntry | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const entry: CacheEntry = JSON.parse(raw);
    if (Date.now() - entry.savedAt > CACHE_TTL_MS) {
      localStorage.removeItem(CACHE_KEY);
      return null;
    }
    return entry;
  } catch {
    return null;
  }
}

export function getCachedFormData(step: string): Record<string, string> | null {
  const entry = getEntry();
  return entry?.data[step] ?? null;
}

export function setCachedFormData(step: string, data: Record<string, string>) {
  try {
    const entry = getEntry() ?? { data: {}, savedAt: Date.now() };
    entry.data[step] = data;
    entry.savedAt = Date.now();
    localStorage.setItem(CACHE_KEY, JSON.stringify(entry));
  } catch {
    // localStorage full or unavailable
  }
}

export function clearStepCache(step: string) {
  try {
    const entry = getEntry();
    if (!entry) return;
    delete entry.data[step];
    localStorage.setItem(CACHE_KEY, JSON.stringify(entry));
  } catch {
    // localStorage unavailable
  }
}

export function clearAllFormCache() {
  try {
    localStorage.removeItem(CACHE_KEY);
  } catch {
    // localStorage unavailable
  }
}
