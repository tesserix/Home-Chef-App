# On-device QA — #62 Customer auth + onboarding E2E

**App:** Home Chef (Customer) · **Build:** TestFlight (customer #5) · **Network:** Wi-Fi/LTE on
**Goal:** every auth path reaches the right place; onboarding completes; session persists; no keychain errors.

For each step: mark **PASS/FAIL**, screenshot. On **FAIL**: copy the exact on-screen text, and if it crashes/hangs, grab logs from **Console.app** (Mac) → filter by the app process while reproducing.

> Tip: use a **fresh email** you can receive (for T1) and a Google + Apple ID not yet used on this app (for T5/T6). If you've used them before, that's fine — note it (existing accounts skip onboarding).

---

## T1 — Email sign-up → onboarding → home
1. Launch app cold → should land on **Sign in** screen.
2. Tap **Sign up**.
3. Enter first name, last name, a fresh email, password (8+ chars), phone → **Sign up**.
4. **Expect:** lands on **onboarding → user-info** (not bounced to Sign in).
5. user-info: fill required fields → continue.
6. address: fill required fields → continue.
7. preferences: set → continue/finish.
8. **Expect:** lands on the **home tab** (chef list). ✅ = #66 signup path verified too.

## T2 — Session persists across cold start
1. Fully close the app (swipe up, kill).
2. Relaunch.
3. **Expect:** opens straight to **home** — no Sign-in screen, no re-onboarding.

## T3 — Sign out
1. Profile tab → **Sign out**.
2. **Expect:** returns to **Sign in** screen.

## T4 — Email login (existing account)
1. From Sign in, enter the T1 email + password → **Sign in**.
2. **Expect:** straight to **home** (onboarding already complete, no wizard).
3. Wrong-password check: sign out, try a wrong password → **Expect:** a *friendly* error ("Wrong password…"), **never** a raw code like `auto_login_401`.

## T5 — Google sign-in
1. Sign out. Tap **Continue with Google** → pick a Google account.
2. **Expect:** **no keychain error** (#66); new Google user → onboarding, existing → home.
3. Profile shows the Google **name + photo**.

## T6 — Apple sign-in (iOS)
1. Sign out. Tap **Continue with Apple** → authenticate.
2. **Expect:** completes; new → onboarding, existing → home. First-time Apple shows your name.

## T7 — Onboarding validation
1. As a fresh user on user-info, try **continue with a required field empty**.
2. **Expect:** blocked, **inline field error** shown (no crash, no silent advance).

## T8 — Profile reflects sign-up data
1. Profile tab.
2. **Expect:** name / email / phone match what you entered (or the Google/Apple identity).

---

## Results
| Test | PASS/FAIL | Notes / exact error text / screenshot # |
|------|-----------|------------------------------------------|
| T1 email signup → home |  |  |
| T2 session persists |  |  |
| T3 sign out |  |  |
| T4 email login (+ wrong-pw msg) |  |  |
| T5 Google sign-in |  |  |
| T6 Apple sign-in |  |  |
| T7 onboarding validation |  |  |
| T8 profile data |  |  |

**Overall:** ___ / 8 passed. Paste this table back to me — I triage any FAIL and ship the fix.
