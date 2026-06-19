package workflows

// order_test.go — the Order lifecycle Saga (#122). Pins the durable state
// machine: the happy path runs notify→accept→ready→dispatch→delivered→settle
// with no refund; and every failure branch (chef rejects, chef-accept timeout,
// cancel before delivery) takes the refund compensation and never dispatches or
// settles. Activities are stubbed via the pluggable Func vars; signals are driven
// with the test environment's delayed callbacks.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// orderSagaStubs swaps the activity transports for counting stubs and restores
// them after the test. Returns pointers to the call counters.
type sagaCounters struct{ notify, dispatch, settle, refund int }

func withOrderSagaStubs(t *testing.T) *sagaCounters {
	t.Helper()
	c := &sagaCounters{}
	on, od, os, orf := NotifyChefFunc, DispatchFunc, OrderSettleFunc, OrderRefundFunc
	t.Cleanup(func() { NotifyChefFunc, DispatchFunc, OrderSettleFunc, OrderRefundFunc = on, od, os, orf })
	NotifyChefFunc = func(context.Context, uuid.UUID) error { c.notify++; return nil }
	DispatchFunc = func(context.Context, uuid.UUID) error { c.dispatch++; return nil }
	OrderSettleFunc = func(context.Context, uuid.UUID) error { c.settle++; return nil }
	OrderRefundFunc = func(context.Context, uuid.UUID, string) error { c.refund++; return nil }
	return c
}

func newOrderSagaEnv() *testsuite.TestWorkflowEnvironment {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(NotifyChefActivity)
	env.RegisterActivity(DispatchDeliveryActivity)
	env.RegisterActivity(OrderSettleActivity)
	env.RegisterActivity(OrderRefundActivity)
	return env
}

func TestOrderSaga_HappyPath(t *testing.T) {
	c := withOrderSagaStubs(t)
	env := newOrderSagaEnv()

	// Drive the signals in lifecycle order.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalChefDecision, ChefDecisionSignal{Accepted: true})
	}, time.Second)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(SignalOrderReady, nil) }, 2*time.Second)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(SignalOrderDelivered, nil) }, 3*time.Second)

	env.ExecuteWorkflow(OrderSagaWorkflow, OrderSagaInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, c.notify)
	require.Equal(t, 1, c.dispatch)
	require.Equal(t, 1, c.settle)
	require.Equal(t, 0, c.refund) // no compensation on the happy path
}

func TestOrderSaga_ChefRejects_Refunds(t *testing.T) {
	c := withOrderSagaStubs(t)
	env := newOrderSagaEnv()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalChefDecision, ChefDecisionSignal{Accepted: false, Reason: "kitchen closed"})
	}, time.Second)

	env.ExecuteWorkflow(OrderSagaWorkflow, OrderSagaInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, c.notify)
	require.Equal(t, 1, c.refund) // compensated
	require.Equal(t, 0, c.dispatch)
	require.Equal(t, 0, c.settle)
}

func TestOrderSaga_ChefAcceptTimeout_Refunds(t *testing.T) {
	c := withOrderSagaStubs(t)
	env := newOrderSagaEnv()

	// No chef decision signal → the accept timer fires (the env fast-forwards it).
	env.ExecuteWorkflow(OrderSagaWorkflow, OrderSagaInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, c.refund)
	require.Equal(t, 0, c.dispatch)
	require.Equal(t, 0, c.settle)
}

func TestOrderSaga_CancelBeforeDelivery_Refunds(t *testing.T) {
	c := withOrderSagaStubs(t)
	env := newOrderSagaEnv()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalChefDecision, ChefDecisionSignal{Accepted: true})
	}, time.Second)
	env.RegisterDelayedCallback(func() { env.SignalWorkflow(SignalOrderReady, nil) }, 2*time.Second)
	// Cancel after dispatch but before delivery.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderCancelled, OrderCancelSignal{Reason: "customer cancelled"})
	}, 3*time.Second)

	env.ExecuteWorkflow(OrderSagaWorkflow, OrderSagaInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, c.dispatch) // got as far as dispatch
	require.Equal(t, 1, c.refund)   // then compensated on cancel
	require.Equal(t, 0, c.settle)   // never settled
}
