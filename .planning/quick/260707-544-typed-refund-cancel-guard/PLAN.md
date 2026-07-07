# #544 — generic full-refund movers must skip typed escrow orders

## Bug
`RefundOrderForCancellation` (services/cancellation_order_refund.go — used by customer `CancelOrder`
in orders.go + chef reject in chefs.go `UpdateOrderStatus`) and the saga `CompensateOrderRefund`
(services/temporal_order.go) issue a GENERIC full refund without the `TypedRefundOrderKind` guard
that `InitiateRefund` already has (#394). A meal-plan-day / group-order-spawned `Order` reaching
these paths would be refunded on the generic `refund:`/`saga-refund:` keyspace — DISJOINT from the
typed `mealplan-refund:`/`grouporder-refund:` keys — while the chef's DIRECT transfer isn't
reversed. Inert today (spawned rows have no gateway payment id, escrow flags OFF) but a real
double-refund the moment a gateway id lands on such a row / the saga is enabled.

## Fix (mirror the #394 InitiateRefund guard)
Both movers: after confirming the order is paid / loaded, call `TypedRefundOrderKind`; if the order
is a typed escrow order, skip the generic refund (return nil, log). Fail safe on a type-check error
(don't refund a possibly-typed order). Naturally-guarded `chef_order_cancel.go` CancelOrder is left
as-is (it 422s on empty `RazorpayPaymentID`, which spawned rows always have).

## Tests (RED-first)
- `RefundOrderForCancellation`: skips a typed meal-plan-day order + a typed group order (no gateway
  call, payment_status stays completed, refund_amount 0); a plain order still refunds.
- `CompensateOrderRefund`: skips a typed order (no wallet credit, order untouched).
- harness: added empty `meal_plan_days`/`group_orders` tables to `setupCancelRefundDB` +
  `addOrdersTable` (TypedRefundOrderKind Counts them).

## Scope note
Closes the MONEY hazard (disjoint-keyspace double-refund). The residual `Order.Status` vs
day/group-status desync on a generic cancel of a spawned order (issue option b — propagate the
cancellation / route to the typed refund flow) is separate, larger, and money-safe with this guard;
noted as a follow-up if warranted. Flag-independent. Adversarial go-developer verify before merge.
