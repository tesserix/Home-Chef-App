# Universal / App Links association files (#58)

These let `https://fe3dr.com/chef/<slug>` open the Fe3dr customer app directly
(iOS Universal Links + Android App Links), so the SEO chef pages deep-link into
the app. The app side is declared in `apps/mobile-customer/app.json`
(`ios.associatedDomains` + the Android `https` intent filter).

**Before this works in production, fill in the real signing identities (placeholders today):**

- `apple-app-site-association` → replace `TEAMID` with the Apple Developer **Team ID**
  (the App ID prefix for `com.tesserix.homechef.customer`). Must be served at
  `https://fe3dr.com/.well-known/apple-app-site-association` with
  `Content-Type: application/json` and no redirect (the nginx rule in
  `nginx.conf` sets the content type).
- `assetlinks.json` → replace `REPLACE_WITH_RELEASE_SIGNING_SHA256` with the
  SHA-256 fingerprint of the **release** signing key for
  `com.tesserix.homechef.customer` (from Play Console → App signing, or
  `keytool -list -v`).

Until then the custom-scheme deep link (`homechef-customer://chef/<slug>`, used by
the landing's "Open in app" CTA) still works for installed apps.
