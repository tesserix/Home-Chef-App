/**
 * Copy a share link to the clipboard. Resolves false rather than throwing, so a
 * failure is a toast instead of a dead screen.
 *
 * `expo-clipboard` is loaded lazily, INSIDE the call, never at module scope
 * (#1038). A JS bundle that reaches for a native module the installed binary
 * predates crashes every screen that merely *imports* the file — Promote
 * redboxed with "Cannot find native module 'ExpoClipboard'" before it drew a
 * single pixel. Loaded here, the link itself still renders and stays readable.
 */
export async function copyLink(url: string): Promise<boolean> {
  if (!url) return false;

  try {
    // require, not import(): both Metro and jest resolve it, and jest cannot run
    // a dynamic import without --experimental-vm-modules.
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const Clipboard = require('expo-clipboard') as {
      setStringAsync: (value: string) => Promise<void>;
    };
    await Clipboard.setStringAsync(url);
    return true;
  } catch {
    return false;
  }
}
