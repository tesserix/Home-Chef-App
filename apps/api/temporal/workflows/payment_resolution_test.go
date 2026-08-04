package workflows

// payment_resolution_test.go — pins the one property that matters: an order is
// cancelled ONLY when the gateway says every attempt is dead. In-flight,
// unknown, and "we ran out of time" must all leave the order alone.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// wire installs the three activity transports and returns counters for them.
func wirePaymentActivities(env *testsuite.TestWorkflowEnvironment, outcomes []PaymentOutcome, err error) (polls, expired, stalled *int) {
	p, e, s := 0, 0, 0
	env.RegisterActivity(ResolvePaymentActivity)
	env.RegisterActivity(ExpireUnpaidOrderActivity)
	env.RegisterActivity(PaymentStalledActivity)

	ResolvePaymentFunc = func(context.Context, uuid.UUID) (PaymentOutcome, error) {
		i := p
		p++
		if err != nil {
			return PaymentOutcomeUnknown, err
		}
		if i >= len(outcomes) {
			return outcomes[len(outcomes)-1], nil
		}
		return outcomes[i], nil
	}
	ExpireUnpaidOrderFunc = func(context.Context, uuid.UUID) error { e++; return nil }
	PaymentStalledFunc = func(context.Context, uuid.UUID, time.Duration) error { s++; return nil }
	return &p, &e, &s
}

// A payment that resolves on the first ask settles and stops.
func TestPaymentResolution_SettlesImmediately(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	polls, expired, _ := wirePaymentActivities(env, []PaymentOutcome{PaymentOutcomeSettled}, nil)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeSettled, res.Outcome)
	require.Equal(t, 1, *polls)
	require.Equal(t, 0, *expired, "a settled payment must never be expired")
}

// The whole point: a payment that sits PENDING and then succeeds must settle,
// never be cancelled on the way.
func TestPaymentResolution_PendingThenSettles_NeverExpires(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	polls, expired, _ := wirePaymentActivities(env, []PaymentOutcome{
		PaymentOutcomeInFlight, PaymentOutcomeInFlight, PaymentOutcomeInFlight, PaymentOutcomeSettled,
	}, nil)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeSettled, res.Outcome)
	require.Equal(t, 4, *polls)
	require.Equal(t, 0, *expired)
}

// Only the gateway's own "every attempt is dead" cancels an order.
func TestPaymentResolution_DeadPayment_Expires(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	_, expired, _ := wirePaymentActivities(env, []PaymentOutcome{
		PaymentOutcomeInFlight, PaymentOutcomeDead,
	}, nil)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeDead, res.Outcome)
	require.Equal(t, 1, *expired)
}

// A payment that never resolves runs to the horizon and hands the order BACK to
// the sweeps still pending. A timer must never be what cancels an order.
func TestPaymentResolution_NeverResolves_HandsBackWithoutCancelling(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	_, expired, stalled := wirePaymentActivities(env, []PaymentOutcome{PaymentOutcomeInFlight}, nil)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeInFlight, res.Outcome,
		"running out of horizon is not an answer — it must not become a cancel")
	require.Equal(t, 0, *expired)
	require.Equal(t, 1, *stalled, "ops is told once, not on every tick")
}

// A gateway that cannot be reached is an UNKNOWN answer, never a dead one. This
// is the branch that would strand real charges if it ever cancelled.
func TestPaymentResolution_GatewayDown_NeverExpires(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	_, expired, _ := wirePaymentActivities(env, nil, errors.New("gateway timeout"))

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(), "an unreachable gateway is not a workflow failure")
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeInFlight, res.Outcome)
	require.Equal(t, 0, *expired)
}

// An order that left the pre-payment world (customer cancelled, already
// refunded) ends the poll cleanly rather than failing the workflow.
func TestPaymentResolution_OrderGone_EndsCleanly(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	_, expired, _ := wirePaymentActivities(env, []PaymentOutcome{PaymentOutcomeGone}, nil)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeGone, res.Outcome)
	require.Equal(t, 0, *expired)
}

// The settled signal wakes the poll early instead of waiting out the interval.
func TestPaymentResolution_SettledSignal_WakesThePollEarly(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	polls, expired, _ := wirePaymentActivities(env, []PaymentOutcome{
		PaymentOutcomeInFlight, PaymentOutcomeSettled,
	}, nil)

	// Fire well inside the first poll interval: the workflow must act on the
	// signal rather than sleep out the timer.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalPaymentResolved, nil)
	}, time.Second)

	env.ExecuteWorkflow(PaymentResolutionWorkflow, PaymentResolutionInput{OrderID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res PaymentResolutionResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, PaymentOutcomeSettled, res.Outcome)
	require.Equal(t, 2, *polls)
	require.Equal(t, 0, *expired)
	require.Less(t, res.Waited, paymentPollFast, "the signal must short-circuit the timer")
}
