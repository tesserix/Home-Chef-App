# Phase 3: Vendor App + Driver Core - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-04-06
**Phase:** 03-Vendor App + Driver Core
**Areas discussed:** Vendor live orders, Vendor menu management, Vendor onboarding, Driver delivery workflow

---

## Vendor Live Orders

All 3 questions (order queue layout, new order alert, accept/reject UX) answered with "You decide" — Claude has full discretion.

## Vendor Menu Management

All 3 questions (edit UX, photo capture, availability toggle) answered with "You decide".

## Vendor Onboarding

Both questions (wizard pattern, document upload) answered with "You decide".

## Driver Delivery Workflow

All 4 questions (available list, active delivery screen, navigation handoff, status updates) answered with "You decide".

---

## Claude's Discretion (all areas)

User explicitly deferred all UX decisions to Claude for this phase. Claude will follow:
- Phase 1 foundation patterns (auth, monorepo, design system)
- Phase 2 mobile UX patterns (bottom sheets, pull-to-refresh, optimistic updates, Zod validation)
- Mobile best practices for kitchen (vendor) and driving (driver) use cases

## Deferred Ideas

- Background GPS + push notifications → Phase 4
- Map pin view for available deliveries → future enhancement
- Chef availability home screen widget → v2 (DIFF-09)
