# Phase 2: Customer App - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 02-Customer App
**Areas discussed:** Home & Chef Discovery, Cart & Checkout, Order Tracking Map, Secondary Screens

---

## Home & Chef Discovery

| Option | Description | Selected |
|--------|-------------|----------|
| Card grid (2 columns) | Photo + name + rating + cuisine, Instagram-like | |
| List with hero images | Full-width cards, more detail per chef | |
| You decide | Claude picks best mobile layout | ✓ |

**User's choice:** You decide (all 3 questions — browse, chef detail, search/filter)

---

## Cart & Checkout

| Option | Description | Selected |
|--------|-------------|----------|
| Bottom sheet cart | Floating bar + swipe up for full cart | |
| Separate cart page | Full cart screen like web | |
| You decide | Claude picks best mobile cart | ✓ |

**Cart UX:** You decide

| Option | Description | Selected |
|--------|-------------|----------|
| Razorpay WebView | Open in expo-web-browser | |
| Razorpay RN SDK | Native payment sheet | ✓ |
| You decide | Claude picks | |

**Payment:** Razorpay React Native SDK

---

## Order Tracking Map

| Option | Description | Selected |
|--------|-------------|----------|
| Polling (Recommended) | Poll every 5s via React Query, no backend changes | ✓ |
| WebSocket | Real-time, needs new backend endpoint | |
| You decide | Claude picks | |

**Tracking approach:** Polling at 5s intervals

| Option | Description | Selected |
|--------|-------------|----------|
| Full screen map | Map fills screen, order details in bottom sheet | |
| Split view | Map top half, details bottom half | |
| You decide | Claude picks best UX | ✓ |

**Map layout:** You decide

---

## Secondary Screens

| Option | Description | Selected |
|--------|-------------|----------|
| Multi-step wizard | Swipeable steps matching web components | |
| Single scrollable form | All fields on one screen | |
| You decide | Claude picks best mobile onboarding | ✓ |

**Onboarding:** You decide

| Option | Description | Selected |
|--------|-------------|----------|
| Same data, mobile UX | Same APIs, redesigned for mobile patterns | |
| Direct port | Match web layout closely | |
| You decide | Claude adapts each screen | ✓ |

**Web parity approach:** You decide

---

## Claude's Discretion

- All browse/discovery layout decisions
- Cart UX pattern
- Map view layout
- Onboarding flow design
- All secondary screen adaptations
- Tab navigation structure

## Deferred Ideas

None
