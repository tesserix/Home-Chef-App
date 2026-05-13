# Mobile Delivery Inventory

Generated: 2026-05-13
App: apps/mobile-delivery/
Total rows: 175

| surface_id | app | category | route_or_file | audience | word_count | current_text_excerpt | shared_component_origin | last_edited | notes |
|---|---|---|---|---|---|---|---|---|---|
| md-meta-001 | mobile-delivery | seo-meta | app.json | driver | 2 | "HomeChef Delivery" | — | 2026-05 | App display name (iOS + Android) |
| md-meta-002 | mobile-delivery | seo-meta | app.json | driver | 6 | "Use Face ID to log in quickly" | native iOS prompt | 2026-05 | NSFaceIDUsageDescription — Face ID rationale |
| md-meta-003 | mobile-delivery | seo-meta | app.json | driver | 3 | "Used to take photos" | native iOS prompt | 2026-05 | NSCameraUsageDescription — VERY terse; P1 candidate for clarification |
| md-meta-004 | mobile-delivery | seo-meta | app.json | driver | 14 | "HomeChef Delivery needs your location to show your position on the delivery map." | native iOS prompt | 2026-05 | NSLocationWhenInUseUsageDescription — safety-critical foreground location |
| md-meta-005 | mobile-delivery | seo-meta | app.json | driver | 16 | "HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status." | native iOS prompt | 2026-05 | NSLocationAlwaysAndWhenInUseUsageDescription — P0: background-location rationale, drift vs in-app modal |
| md-meta-006 | mobile-delivery | seo-meta | app.json | driver | 13 | "HomeChef Delivery tracks your location during deliveries to keep customers updated." | native Android prompt | 2026-05 | expo-location plugin locationAlwaysAndWhenInUsePermission — slightly different wording from iOS Info.plist (drift) |
| md-auth-001 | mobile-delivery | auth-onboarding | app/(auth)/login.tsx | driver | 2 | "Welcome back" | LoginScreen prop (mobile-shared) | 2026-05 | Title override; shared LoginScreen renders rest |
| md-auth-002 | mobile-delivery | auth-onboarding | app/(auth)/login.tsx | driver | 6 | error: "Google sign-in failed: no ID token" | thrown Error | 2026-05 | Developer-shaped error; surfaces to user if uncaught |
| md-auth-003 | mobile-delivery | auth-onboarding | app/(auth)/login.tsx | driver | 6 | error: "Apple sign-in failed: no identity token" | thrown Error | 2026-05 | Developer-shaped error |
| md-auth-004 | mobile-delivery | auth-onboarding | app/(auth)/login.tsx | driver | 3 | error: "Biometric authentication failed" | thrown Error | 2026-05 | Surfaced if biometric fails |
| md-auth-005 | mobile-delivery | auth-onboarding | app/(auth)/login.tsx | driver | 8 | error: "No saved session found. Please log in with email." | thrown Error | 2026-05 | Recovery hint after biometric-no-session edge case |
| md-onb-001 | mobile-delivery | auth-onboarding | app/(onboarding)/_layout.tsx | driver | 4 | "Step 1 of 6" | — | 2026-05 | Header stepper text — repeated 1..6 |
| md-onb-002 | mobile-delivery | auth-onboarding | app/(onboarding)/_layout.tsx | driver | 2 | "Driver Onboarding" | — | 2026-05 | Header fallback title |
| md-onb-003 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 2 | "Personal Information" | — | 2026-05 | Screen heading |
| md-onb-004 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 7 | "Tell us about yourself to get started" | — | 2026-05 | Subheading |
| md-onb-005 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 1 | "City" | — | 2026-05 | Field label |
| md-onb-006 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 3 | "Enter your city" | — | 2026-05 | TextInput placeholder |
| md-onb-007 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 6 | error: "City must be at least 2 characters" | zod validation | 2026-05 | Zod error message |
| md-onb-008 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 2 | "Vehicle Type" | — | 2026-05 | Field label |
| md-onb-009 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 1 | "Bike" / "Scooter" / "Car" / "Van" | — | 2026-05 | Vehicle type chip labels (4 entries, each <3 words but kept as set) |
| md-onb-010 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 3 | "Emergency Contact Name" | — | 2026-05 | Field label |
| md-onb-011 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 5 | "Full name of emergency contact" | — | 2026-05 | TextInput placeholder |
| md-onb-012 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 5 | error: "Emergency contact name is required" | zod validation | 2026-05 | Validation error |
| md-onb-013 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 3 | "Emergency Contact Phone" | — | 2026-05 | Field label |
| md-onb-014 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 3 | "10-digit mobile number" | — | 2026-05 | TextInput placeholder |
| md-onb-015 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 7 | error: "Enter a valid 10-digit Indian mobile number" | zod validation | 2026-05 | Validation error — India-specific |
| md-onb-016 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 3 | "Date of Birth" | — | 2026-05 | Field label |
| md-onb-017 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 1 | "(optional)" | — | 2026-05 | Field qualifier (kept due to context) |
| md-onb-018 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 1 | "MM/DD/YYYY" | — | 2026-05 | Placeholder — INCONSISTENT with India-only validation elsewhere (P2 drift) |
| md-onb-019 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 1 | "Next" | — | 2026-05 | Primary CTA |
| md-onb-020 | mobile-delivery | auth-onboarding | app/(onboarding)/personal.tsx | driver | 7 | error: "Failed to save personal info. Please try again." | catch fallback | 2026-05 | Generic API error fallback |
| md-onb-021 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle Details" | — | 2026-05 | Screen heading |
| md-onb-022 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle type:" | — | 2026-05 | Subheading |
| md-onb-023 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle Make" | — | 2026-05 | Field label |
| md-onb-024 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "e.g. Honda" | — | 2026-05 | Placeholder |
| md-onb-025 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 4 | error: "Vehicle make is required" | zod validation | 2026-05 | Validation error |
| md-onb-026 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle Model" | — | 2026-05 | Field label |
| md-onb-027 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 3 | "e.g. Activa 6G" | — | 2026-05 | Placeholder (India-specific) |
| md-onb-028 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 4 | error: "Vehicle model is required" | zod validation | 2026-05 | Validation error |
| md-onb-029 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle Year" | — | 2026-05 | Field label |
| md-onb-030 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 5 | error: "Enter a valid 4-digit year" | zod validation | 2026-05 | Validation error |
| md-onb-031 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 6 | error: "Year must be between 2000 and {currentYear}" | zod validation | 2026-05 | Range validation |
| md-onb-032 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "Vehicle Color" | — | 2026-05 | Field label |
| md-onb-033 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "e.g. Black" | — | 2026-05 | Placeholder |
| md-onb-034 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 4 | error: "Vehicle color is required" | zod validation | 2026-05 | Validation error |
| md-onb-035 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 3 | "Vehicle Registration Number" | — | 2026-05 | Field label |
| md-onb-036 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 2 | "e.g. MH12AB1234" | — | 2026-05 | Placeholder (Indian format) |
| md-onb-037 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 8 | error: "Enter a valid vehicle number (e.g. MH12AB1234)" | zod validation | 2026-05 | Validation error |
| md-onb-038 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 3 | "Driving License Number" | — | 2026-05 | Field label |
| md-onb-039 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 3 | "Min 8 characters" | — | 2026-05 | Placeholder |
| md-onb-040 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 7 | error: "License number must be at least 8 characters" | zod validation | 2026-05 | Validation error |
| md-onb-041 | mobile-delivery | auth-onboarding | app/(onboarding)/vehicle.tsx | driver | 9 | error: "Failed to save vehicle details. Please try again." | catch fallback | 2026-05 | Generic API error fallback |
| md-onb-042 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "Upload Documents" | — | 2026-05 | Screen heading |
| md-onb-043 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 9 | "Please upload clear photos or PDFs of your documents" | — | 2026-05 | Subheading |
| md-onb-044 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "Driving License" | — | 2026-05 | Slot label |
| md-onb-045 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "ID Proof" | — | 2026-05 | Slot label |
| md-onb-046 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 3 | "Vehicle RC (optional)" | — | 2026-05 | Slot label (India-specific term — RC = Registration Certificate) |
| md-onb-047 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "PDF uploaded" | — | 2026-05 | Preview label |
| md-onb-048 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 3 | "No document uploaded" | — | 2026-05 | Empty preview placeholder |
| md-onb-049 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 1 | "Uploading..." | — | 2026-05 | Loading state |
| md-onb-050 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 1 | "Camera" | — | 2026-05 | Action button (kept as set) |
| md-onb-051 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 1 | "Gallery" | — | 2026-05 | Action button (kept as set) |
| md-onb-052 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "Upload PDF" | — | 2026-05 | Action button |
| md-onb-053 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "Permission Required" | Alert title | 2026-05 | Camera permission gate |
| md-onb-054 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 7 | "Camera permission is required to capture documents." | Alert body | 2026-05 | Camera permission rationale |
| md-onb-055 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 1 | "Upload Error" | Alert title | 2026-05 | Error title |
| md-onb-056 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 4 | "Upload failed. Please try again." | catch fallback | 2026-05 | Generic upload failure |
| md-onb-057 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 2 | "Required Documents" | Alert title | 2026-05 | Gate alert |
| md-onb-058 | mobile-delivery | auth-onboarding | app/(onboarding)/documents.tsx | driver | 8 | "Please upload Driving License and ID Proof to continue." | Alert body | 2026-05 | Required-docs gating message |
| md-onb-059 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "Payout Details" | — | 2026-05 | Screen heading |
| md-onb-060 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 8 | "Choose how you would like to receive your earnings" | — | 2026-05 | Subheading |
| md-onb-061 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "Bank Account" | — | 2026-05 | Tab label |
| md-onb-062 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 1 | "UPI" | — | 2026-05 | Tab label (India-specific) |
| md-onb-063 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "Account Number" | — | 2026-05 | Field label |
| md-onb-064 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 3 | "Enter account number" | — | 2026-05 | Placeholder |
| md-onb-065 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 3 | "Confirm Account Number" | — | 2026-05 | Field label |
| md-onb-066 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 3 | "Re-enter account number" | — | 2026-05 | Placeholder |
| md-onb-067 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 4 | error: "Please confirm account number" | zod | 2026-05 | Validation |
| md-onb-068 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 4 | error: "Account numbers do not match" | zod | 2026-05 | Validation |
| md-onb-069 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 4 | error: "Account number is required" | zod | 2026-05 | Validation |
| md-onb-070 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "IFSC Code" | — | 2026-05 | Field label (India-specific) |
| md-onb-071 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "e.g. HDFC0001234" | — | 2026-05 | Placeholder |
| md-onb-072 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 8 | error: "Enter a valid IFSC code (e.g. HDFC0001234)" | zod | 2026-05 | Validation |
| md-onb-073 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "Bank Name" | — | 2026-05 | Field label |
| md-onb-074 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 3 | "e.g. HDFC Bank" | — | 2026-05 | Placeholder |
| md-onb-075 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 4 | error: "Bank name is required" | zod | 2026-05 | Validation |
| md-onb-076 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "UPI ID" | — | 2026-05 | Field label |
| md-onb-077 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 3 | "e.g. yourname@upi" | — | 2026-05 | Placeholder |
| md-onb-078 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 7 | error: "Enter a valid UPI ID (must contain @)" | zod | 2026-05 | Validation |
| md-onb-079 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 19 | "Your payout details are encrypted and never stored on this device. They are used solely for processing your earnings." | — | 2026-05 | Trust/security copy |
| md-onb-080 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 2 | "Select a Plan" | Alert title | 2026-05 | Used by Subscription only? No — Alert.alert in payout? Actually appears in subscription screen |
| md-onb-081 | mobile-delivery | auth-onboarding | app/(onboarding)/payout.tsx | driver | 8 | error: "Failed to save payout details. Please try again." | catch fallback | 2026-05 | Generic API failure |
| md-onb-082 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 3 | "Choose Your Plan" | — | 2026-05 | Screen heading |
| md-onb-083 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 8 | "Select a subscription plan that works best for you" | — | 2026-05 | Subheading |
| md-onb-084 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 1 | "Recommended" | — | 2026-05 | Plan badge |
| md-onb-085 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 1 | "/month" | — | 2026-05 | Price suffix (kept due to currency context) |
| md-onb-086 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 4 | "Up to {n} deliveries/month" | — | 2026-05 | Plan capacity line |
| md-onb-087 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 2 | "Select Plan" | — | 2026-05 | Primary CTA |
| md-onb-088 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 2 | "Select a Plan" | Alert title | 2026-05 | Gating alert title |
| md-onb-089 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 9 | "Please select a subscription plan to continue." | Alert body | 2026-05 | Gating alert body |
| md-onb-090 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 4 | "Failed to load subscription plans." | empty/error state | 2026-05 | Error state |
| md-onb-091 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 1 | "Retry" | — | 2026-05 | Retry CTA |
| md-onb-092 | mobile-delivery | auth-onboarding | app/(onboarding)/subscription.tsx | driver | 7 | error: "Failed to select plan. Please try again." | catch fallback | 2026-05 | Generic API failure |
| md-onb-093 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 3 | "Review Your Application" | — | 2026-05 | Screen heading |
| md-onb-094 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 6 | "Please review your details before submitting" | — | 2026-05 | Subheading |
| md-onb-095 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Personal Info" | — | 2026-05 | Section header |
| md-onb-096 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Vehicle Details" | — | 2026-05 | Section header |
| md-onb-097 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 1 | "Documents" | — | 2026-05 | Section header |
| md-onb-098 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 3 | "Documents Uploaded" | — | 2026-05 | Summary row label |
| md-onb-099 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 4 | "{n}/{total} required + RC" | — | 2026-05 | Summary row value pattern |
| md-onb-100 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Payout Method" | — | 2026-05 | Section header |
| md-onb-101 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Subscription Plan" | — | 2026-05 | Section header |
| md-onb-102 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Selected Plan" | — | 2026-05 | Row label |
| md-onb-103 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 9 | "I accept the Terms of Service and Privacy Policy" | — | 2026-05 | Legal consent |
| md-leg-001 | mobile-delivery | legal | app/(onboarding)/review.tsx | driver | 3 | "Terms of Service" | — | 2026-05 | Legal link (no navigation wired in code) |
| md-leg-002 | mobile-delivery | legal | app/(onboarding)/review.tsx | driver | 2 | "Privacy Policy" | — | 2026-05 | Legal link |
| md-onb-104 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Terms Required" | Alert title | 2026-05 | Gate alert |
| md-onb-105 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 7 | "Please accept the Terms of Service and Privacy Policy." | Alert body | 2026-05 | Gate alert body |
| md-onb-106 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Submit Application" | — | 2026-05 | Primary CTA |
| md-onb-107 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 2 | "Submission Error" | Alert title | 2026-05 | Failure alert |
| md-onb-108 | mobile-delivery | auth-onboarding | app/(onboarding)/review.tsx | driver | 7 | "Failed to submit application. Please try again." | catch fallback | 2026-05 | Generic API failure |
| md-onb-109 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 3 | "Application Not Approved" | — | 2026-05 | Rejected state heading |
| md-onb-110 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 9 | "Unfortunately your application was not approved at this time." | — | 2026-05 | Rejected state body |
| md-onb-111 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 1 | "Reason:" | — | 2026-05 | Rejection-reason label |
| md-onb-112 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 1 | "Reapply" | — | 2026-05 | CTA |
| md-onb-113 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 3 | "Application Under Review" | — | 2026-05 | Pending state heading |
| md-onb-114 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 11 | "Your application has been submitted and is being reviewed by our team." | — | 2026-05 | Pending state body |
| md-onb-115 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 5 | "Estimated review time: 24–48 hours" | — | 2026-05 | SLA promise |
| md-onb-116 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 17 | "We'll notify you once your application is approved. This page checks for updates automatically every 30 seconds." | — | 2026-05 | Reassurance |
| md-onb-117 | mobile-delivery | auth-onboarding | app/(onboarding)/pending.tsx | driver | 1 | "Logout" | — | 2026-05 | Header action |
| md-core-001 | mobile-delivery | core-ux | app/(tabs)/_layout.tsx | driver | 1 | "Dashboard" | — | 2026-05 | Tab label (kept as set) |
| md-core-002 | mobile-delivery | core-ux | app/(tabs)/_layout.tsx | driver | 1 | "Available" | — | 2026-05 | Tab label |
| md-core-003 | mobile-delivery | core-ux | app/(tabs)/_layout.tsx | driver | 1 | "Active" | — | 2026-05 | Tab label |
| md-core-004 | mobile-delivery | core-ux | app/(tabs)/_layout.tsx | driver | 1 | "More" | — | 2026-05 | Tab label |
| md-core-005 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 1 | "Dashboard" | — | 2026-05 | Screen heading |
| md-core-006 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 3 | "You are Online" | — | 2026-05 | Status banner — P0 SAFETY: glanceable, OK |
| md-core-007 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 3 | "You are Offline" | — | 2026-05 | Status banner |
| md-core-008 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 3 | "Receiving delivery requests" | — | 2026-05 | Online subtitle |
| md-core-009 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 6 | "Toggle to start receiving requests" | — | 2026-05 | Offline subtitle |
| md-core-010 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 1 | "Today" / "Week" / "Month" | — | 2026-05 | Period selector — kept as set |
| md-core-011 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 1 | "Deliveries" | — | 2026-05 | Stat card label |
| md-core-012 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 1 | "Earnings" | — | 2026-05 | Stat card label |
| md-core-013 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 1 | "Rating" | — | 2026-05 | Stat card label |
| md-core-014 | mobile-delivery | core-ux | app/(tabs)/index.tsx | driver | 2 | "Total Deliveries" | — | 2026-05 | Stat card label |
| md-err-001 | mobile-delivery | errors-empty | app/(tabs)/index.tsx | driver | 4 | "Failed to load dashboard" | — | 2026-05 | Error empty state |
| md-core-015 | mobile-delivery | core-ux | app/(tabs)/available.tsx | driver | 1 | "Available" | — | 2026-05 | Screen heading |
| md-core-016 | mobile-delivery | core-ux | app/(tabs)/available.tsx | driver | 2 | "You're Offline" | — | 2026-05 | Offline empty state heading |
| md-core-017 | mobile-delivery | core-ux | app/(tabs)/available.tsx | driver | 8 | "Go online to see available deliveries near you" | — | 2026-05 | Offline empty state body |
| md-core-018 | mobile-delivery | core-ux | app/(tabs)/available.tsx | driver | 6 | "Go Online to Accept Deliveries" | — | 2026-05 | P0 SAFETY: "Go Online" CTA — clear enough |
| md-err-002 | mobile-delivery | errors-empty | app/(tabs)/available.tsx | driver | 7 | "No deliveries available nearby. Pull to refresh." | — | 2026-05 | Empty state |
| md-core-019 | mobile-delivery | core-ux | components/driver/DeliveryCard.tsx | driver | 3 | "{n} km away" | — | 2026-05 | Distance pattern — P0 SAFETY |
| md-core-020 | mobile-delivery | core-ux | components/driver/DeliveryCard.tsx | driver | 1 | "items" / "item" | — | 2026-05 | Singular/plural item label |
| md-core-021 | mobile-delivery | core-ux | components/driver/DeliveryCard.tsx | driver | 3 | "~{n} min" | — | 2026-05 | ETA pattern — P0 SAFETY |
| md-core-022 | mobile-delivery | core-ux | components/driver/DeliveryCard.tsx | driver | 2 | "Accept Delivery" | — | 2026-05 | Primary CTA — P0 SAFETY: irreversible action, clear |
| md-core-023 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Active Delivery" | — | 2026-05 | Screen heading |
| md-core-024 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Order #{shortId}" | — | 2026-05 | Order ID pattern |
| md-core-025 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Head to Kitchen" | — | 2026-05 | P0 SAFETY: status label for 'assigned' — directional verb |
| md-core-026 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Waiting for Order" | — | 2026-05 | P0 SAFETY: status label for 'at_pickup' |
| md-core-027 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Order Picked Up" | — | 2026-05 | P0 SAFETY: status label for 'picked_up' |
| md-core-028 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "On the Way" | — | 2026-05 | P0 SAFETY: status label for 'in_transit' |
| md-core-029 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "At Dropoff Location" | — | 2026-05 | P0 SAFETY: status label for 'at_dropoff' |
| md-core-030 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Delivery Complete" | — | 2026-05 | Terminal status label |
| md-core-031 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Delivery Cancelled" | — | 2026-05 | Terminal status label (also banner) |
| md-core-032 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Arrived at Kitchen" | — | 2026-05 | P0 SAFETY: action label — slide-to-confirm verb (assigned → at_pickup) |
| md-core-033 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Picked Up Order" | — | 2026-05 | P0 SAFETY: irreversible — chef hands over food. "Picked Up" past-tense may be confusing as a future action (P1 candidate for review) |
| md-core-034 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Start Delivery" | — | 2026-05 | P0 SAFETY: action label (picked_up → in_transit) |
| md-core-035 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Arrived at Dropoff" | — | 2026-05 | P0 SAFETY: action label (in_transit → at_dropoff) |
| md-core-036 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 4 | "Mark as Delivered" | — | 2026-05 | P0 SAFETY: terminal action — customer-facing consequence |
| md-core-037 | mobile-delivery | core-ux | components/driver/SlideToConfirm.tsx | driver | 3 | "Slide to {label}" | — | 2026-05 | P0 SAFETY: confirmation gesture prompt — composed with action labels above |
| md-core-038 | mobile-delivery | core-ux | components/driver/StatusStepIndicator.tsx | driver | 2 | "Picked Up" / "In Transit" / "At Dropoff" / "Delivered" | — | 2026-05 | Step indicator labels (set of 4) |
| md-core-039 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 11 | "This delivery has been cancelled. Check the Available tab for new requests." | — | 2026-05 | Cancelled banner copy |
| md-core-040 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Pickup Location" | — | 2026-05 | Card header — P0 SAFETY |
| md-core-041 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Dropoff Location" | — | 2026-05 | Card header — P0 SAFETY |
| md-core-042 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 1 | "Note:" | — | 2026-05 | Instruction prefix (pickup/dropoff) |
| md-core-043 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 1 | "Navigate" | — | 2026-05 | P0 SAFETY: navigation CTA — launches Apple/Google Maps. Short but clear |
| md-core-044 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 3 | "Order Summary ({n} item(s))" | — | 2026-05 | Collapsible header pattern |
| md-core-045 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 1 | "Total" | — | 2026-05 | Order summary row |
| md-core-046 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 2 | "Special instructions:" | — | 2026-05 | Order note prefix |
| md-core-047 | mobile-delivery | core-ux | app/(tabs)/active.tsx | driver | 7 | "You earned ₹{payout} for this delivery" | — | 2026-05 | Success/celebration |
| md-emp-001 | mobile-delivery | errors-empty | app/(tabs)/active.tsx | driver | 3 | "No Active Delivery" | — | 2026-05 | Empty state heading |
| md-emp-002 | mobile-delivery | errors-empty | app/(tabs)/active.tsx | driver | 10 | "Accept a delivery from the Available tab to get started." | — | 2026-05 | Empty state body |
| md-trx-001 | mobile-delivery | transactional | components/LocationRationaleModal.tsx | driver | 3 | "Background Location Needed" | — | 2026-05 | P0 SAFETY: pre-OS-prompt rationale modal title |
| md-trx-002 | mobile-delivery | transactional | components/LocationRationaleModal.tsx | driver | 31 | "To keep customers updated on their delivery, HomeChef Delivery needs to track your location while you are on an active delivery — even when the app is in the background." | — | 2026-05 | P0 SAFETY: explains why background location needed; longer than iOS Info.plist string (drift) |
| md-trx-003 | mobile-delivery | transactional | components/LocationRationaleModal.tsx | driver | 18 | "Location tracking stops automatically when the delivery is completed. Battery usage is minimised by only tracking every 15 seconds." | — | 2026-05 | Battery/privacy reassurance |
| md-trx-004 | mobile-delivery | transactional | components/LocationRationaleModal.tsx | driver | 3 | "Allow Background Location" | — | 2026-05 | P0 SAFETY: CTA — clear |
| md-trx-005 | mobile-delivery | transactional | components/LocationRationaleModal.tsx | driver | 2 | "Not Now" | — | 2026-05 | Deny CTA — soft |
| md-core-048 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "More" | — | 2026-05 | Screen heading |
| md-core-049 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "Profile" | — | 2026-05 | Nav item |
| md-core-050 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "Earnings" | — | 2026-05 | Nav item |
| md-core-051 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "History" | — | 2026-05 | Nav item |
| md-core-052 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "Fleet" | — | 2026-05 | Nav item |
| md-core-053 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "Staff" | — | 2026-05 | Nav item |
| md-core-054 | mobile-delivery | core-ux | app/(tabs)/more.tsx | driver | 1 | "Settings" | — | 2026-05 | Nav item |
| md-trx-006 | mobile-delivery | transactional | app/(tabs)/more.tsx | driver | 1 | "Logout" | Alert title | 2026-05 | Logout confirmation title |
| md-trx-007 | mobile-delivery | transactional | app/(tabs)/more.tsx | driver | 6 | "Are you sure you want to logout?" | Alert body | 2026-05 | Logout confirmation body (also appears in pending screen) |
| md-mic-001 | mobile-delivery | microcopy | app/(tabs)/more.tsx | driver | 1 | "Cancel" / "Logout" | Alert buttons | 2026-05 | Standard Alert buttons (kept as set) |
| md-core-055 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 1 | "Settings" | — | 2026-05 | Screen heading |
| md-core-056 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 1 | "Notifications" | — | 2026-05 | Section header |
| md-core-057 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 3 | "New Delivery Notifications" | — | 2026-05 | Toggle label |
| md-core-058 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 7 | "Get notified when a new delivery is available" | — | 2026-05 | Toggle description |
| md-core-059 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 3 | "Earnings Payout Notifications" | — | 2026-05 | Toggle label |
| md-core-060 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 7 | "Get notified when a payout is processed" | — | 2026-05 | Toggle description |
| md-core-061 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 1 | "Availability" | — | 2026-05 | Section header |
| md-core-062 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 3 | "Default Online Status" | — | 2026-05 | Toggle label — P0 SAFETY consideration: auto-online when app opens. Drivers should understand consequence |
| md-core-063 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 6 | "Automatically go online when app opens" | — | 2026-05 | Toggle description |
| md-core-064 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 1 | "Account" | — | 2026-05 | Section header |
| md-core-065 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 2 | "Change Password" | — | 2026-05 | Action row |
| md-core-066 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 3 | "View Subscription Plan" | — | 2026-05 | Action row |
| md-trx-008 | mobile-delivery | transactional | app/driver-settings.tsx | driver | 2 | "Subscription" | Alert title | 2026-05 | Alert title |
| md-trx-009 | mobile-delivery | transactional | app/driver-settings.tsx | driver | 8 | "Visit the web portal to manage your subscription." | Alert body | 2026-05 | Web fallback — cross-app dependency |
| md-core-067 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 2 | "Delete Account" | — | 2026-05 | Destructive action |
| md-trx-010 | mobile-delivery | transactional | app/driver-settings.tsx | driver | 9 | "Contact support at support@homechef.in to request account deletion." | Alert body | 2026-05 | Support email hardcoded — India domain |
| md-core-068 | mobile-delivery | core-ux | app/driver-settings.tsx | driver | 2 | "App Version" | — | 2026-05 | Static footer label |
| md-core-069 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 1 | "Earnings" | — | 2026-05 | Screen heading |
| md-core-070 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 2 | "{Period} Earnings" | — | 2026-05 | Period card heading pattern |
| md-core-071 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 2 | "Total Earned" | — | 2026-05 | Stat |
| md-core-072 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 2 | "Pending Payout" | — | 2026-05 | Stat |
| md-core-073 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 2 | "Last Payout" | — | 2026-05 | Card heading |
| md-core-074 | mobile-delivery | core-ux | app/driver-earnings.tsx | driver | 2 | "Weekly History" | — | 2026-05 | Chart heading |
| md-err-003 | mobile-delivery | errors-empty | app/driver-earnings.tsx | driver | 3 | "Failed to load earnings" | — | 2026-05 | Error state |
| md-core-075 | mobile-delivery | core-ux | app/driver-history.tsx | driver | 2 | "Delivery History" | — | 2026-05 | Screen heading |
| md-emp-003 | mobile-delivery | errors-empty | app/driver-history.tsx | driver | 9 | "No deliveries yet. Your completed deliveries will appear here." | — | 2026-05 | Empty state |
| md-err-004 | mobile-delivery | errors-empty | app/driver-history.tsx | driver | 4 | "Failed to load delivery history" | — | 2026-05 | Error state |
| md-core-076 | mobile-delivery | core-ux | app/delivery/[id].tsx | driver | 2 | "Delivery Detail" | — | 2026-05 | Screen heading |
| md-mic-002 | mobile-delivery | microcopy | app/delivery/[id].tsx | driver | 2 | "Go back" | accessibilityLabel | 2026-05 | a11y back button label |
| md-core-077 | mobile-delivery | core-ux | app/delivery/[id].tsx | driver | 1 | "Route" | — | 2026-05 | Section header |
| md-core-078 | mobile-delivery | core-ux | app/delivery/[id].tsx | driver | 1 | "Pickup" | — | 2026-05 | Row label (kept as set with Drop-off) |
| md-core-079 | mobile-delivery | core-ux | app/delivery/[id].tsx | driver | 1 | "Drop-off" | — | 2026-05 | Row label — DRIFT: 'Drop-off' here vs 'Dropoff' elsewhere |
| md-core-080 | mobile-delivery | core-ux | app/delivery/[id].tsx | driver | 1 | "Distance" / "Completed" / "Payout" | — | 2026-05 | Detail row labels (set) |
| md-err-005 | mobile-delivery | errors-empty | app/delivery/[id].tsx | driver | 4 | "Failed to load delivery detail" | — | 2026-05 | Error state |
| md-emp-004 | mobile-delivery | errors-empty | app/delivery/[id].tsx | driver | 2 | "Delivery not found." | — | 2026-05 | Empty state |
| md-core-081 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 1 | "Profile" | — | 2026-05 | Screen heading |
| md-core-082 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Personal Info" | — | 2026-05 | Section header |
| md-core-083 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Full Name" | — | 2026-05 | Field label |
| md-core-084 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 1 | "Phone" / "City" / "Email" | — | 2026-05 | Field labels (set) |
| md-core-085 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Vehicle Details" | — | 2026-05 | Section header |
| md-core-086 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Vehicle Type" | — | 2026-05 | Field label |
| md-core-087 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Registration Number" | — | 2026-05 | Field label |
| md-core-088 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 1 | "Account" | — | 2026-05 | Section header |
| md-core-089 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "Save Changes" | — | 2026-05 | Primary CTA |
| md-core-090 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 1 | "Edit" / "Cancel" | — | 2026-05 | Action buttons (set) |
| md-core-091 | mobile-delivery | core-ux | app/driver-profile.tsx | driver | 2 | "✓ Verified" / "Pending" | — | 2026-05 | Status badge values |
| md-trx-011 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 1 | "Validation" | Alert title | 2026-05 | Validation alert title |
| md-trx-012 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 5 | "Name, phone, and city are required." | Alert body | 2026-05 | Validation message |
| md-trx-013 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 1 | "Success" | Alert title | 2026-05 | Success alert title |
| md-trx-014 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 3 | "Profile updated successfully." | Alert body | 2026-05 | Success message |
| md-trx-015 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 6 | "Failed to update profile. Please try again." | Alert body | 2026-05 | Generic failure |
| md-trx-016 | mobile-delivery | transactional | app/driver-profile.tsx | driver | 6 | "Failed to upload photo. Please try again." | Alert body | 2026-05 | Photo upload failure |
| md-err-006 | mobile-delivery | errors-empty | app/driver-profile.tsx | driver | 3 | "Failed to load profile" | — | 2026-05 | Error state |
| md-core-092 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 2 | "Fleet Overview" | — | 2026-05 | Screen heading |
| md-core-093 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 2 | "Total Drivers" | — | 2026-05 | Stat |
| md-core-094 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 2 | "Online Now" | — | 2026-05 | Stat |
| md-core-095 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 2 | "Today's Deliveries" | — | 2026-05 | Stat |
| md-core-096 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 2 | "Today's Earnings" | — | 2026-05 | Stat |
| md-core-097 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 1 | "Partners" | — | 2026-05 | Section header |
| md-core-098 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 1 | "Active" / "Inactive" | — | 2026-05 | Partner status badges |
| md-core-099 | mobile-delivery | core-ux | app/fleet/index.tsx | driver | 3 | "Today: {n} deliveries" | — | 2026-05 | Partner row stat pattern |
| md-emp-005 | mobile-delivery | errors-empty | app/fleet/index.tsx | driver | 3 | "No partners found." | — | 2026-05 | Empty state |
| md-err-007 | mobile-delivery | errors-empty | app/fleet/index.tsx | driver | 4 | "Failed to load fleet data" | — | 2026-05 | Error state |
| md-emp-006 | mobile-delivery | errors-empty | app/fleet/index.tsx | driver | 2 | "Fleet Management" | — | 2026-05 | Lock-screen heading |
| md-emp-007 | mobile-delivery | errors-empty | app/fleet/index.tsx | driver | 14 | "Fleet management is available for fleet managers only. Contact your administrator to request access." | — | 2026-05 | Locked-feature explanation |
| md-core-100 | mobile-delivery | core-ux | app/fleet/partner/[id].tsx | driver | 2 | "Partner Detail" | — | 2026-05 | Screen heading |
| md-emp-008 | mobile-delivery | errors-empty | app/fleet/partner/[id].tsx | driver | 3 | "Partner not found." | — | 2026-05 | Empty state |
| md-err-008 | mobile-delivery | errors-empty | app/fleet/partner/[id].tsx | driver | 4 | "Failed to load partner detail" | — | 2026-05 | Error state |
| md-core-101 | mobile-delivery | core-ux | app/staff.tsx | driver | 1 | "Staff" | — | 2026-05 | Screen heading |
| md-core-102 | mobile-delivery | core-ux | app/staff.tsx | driver | 1 | "Invite" | — | 2026-05 | Action button |
| md-core-103 | mobile-delivery | core-ux | app/staff.tsx | driver | 3 | "Invite Staff Member" | — | 2026-05 | Modal heading |
| md-core-104 | mobile-delivery | core-ux | app/staff.tsx | driver | 3 | "colleague@example.com" | — | 2026-05 | Email placeholder |
| md-core-105 | mobile-delivery | core-ux | app/staff.tsx | driver | 5 | error: "Enter a valid email address" | zod | 2026-05 | Validation |
| md-core-106 | mobile-delivery | core-ux | app/staff.tsx | driver | 3 | "e.g. manager, coordinator" | — | 2026-05 | Role placeholder |
| md-core-107 | mobile-delivery | core-ux | app/staff.tsx | driver | 3 | error: "Role is required" | zod | 2026-05 | Validation |
| md-core-108 | mobile-delivery | core-ux | app/staff.tsx | driver | 2 | "Send Invitation" | — | 2026-05 | Primary CTA |
| md-core-109 | mobile-delivery | core-ux | app/staff.tsx | driver | 2 | "Joined {date}" | — | 2026-05 | Staff card subline |
| md-trx-017 | mobile-delivery | transactional | app/staff.tsx | driver | 4 | "Invitation sent to {email}" | Alert body | 2026-05 | Success |
| md-trx-018 | mobile-delivery | transactional | app/staff.tsx | driver | 7 | "You do not have permission to invite staff." | Alert body | 2026-05 | 403 error |
| md-trx-019 | mobile-delivery | transactional | app/staff.tsx | driver | 6 | "Failed to send invitation. Please try again." | Alert body | 2026-05 | Generic failure |
| md-emp-009 | mobile-delivery | errors-empty | app/staff.tsx | driver | 3 | "No staff members yet." | — | 2026-05 | Empty state |
| md-emp-010 | mobile-delivery | errors-empty | app/staff.tsx | driver | 2 | "Staff Management" | — | 2026-05 | Lock-screen heading |
| md-emp-011 | mobile-delivery | errors-empty | app/staff.tsx | driver | 14 | "Staff management requires manager permissions. Contact your administrator to request access." | — | 2026-05 | Locked-feature explanation |
| md-err-009 | mobile-delivery | errors-empty | app/staff.tsx | driver | 4 | "Failed to load staff list" | — | 2026-05 | Error state |
| md-mic-003 | mobile-delivery | microcopy | app/_layout.tsx | driver | 2 | "New Deliveries" | — | 2026-05 | Android notification channel name — P0 SAFETY: user-visible in system settings |
| md-mic-004 | mobile-delivery | microcopy | app/_layout.tsx | driver | 2 | "Delivery Updates" | — | 2026-05 | Android notification channel name |
| md-trx-020 | mobile-delivery | transactional | (shared) OfflineBanner | driver | — | (shared origin) | @homechef/mobile-shared | 2026-05 | OfflineBanner imported — copy lives in mobile-shared |
