package services

import (
	"context"
	"fmt"
	"log"
	"time"

	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// cronJob is one registered scheduled job: a stable name, how often it runs, and
// the run-once work (the existing in-process scan). When Temporal is enabled
// these become durable Temporal Schedules (exactly-once, leader-elected, no
// missed windows); otherwise the legacy in-process tickers run (StartCronJobs).
type cronJob struct {
	name     string
	interval time.Duration
	run      func(context.Context)
	ticker   func(context.Context) // legacy in-process fallback
}

func cronJobs() []cronJob {
	return []cronJob{
		{"weekly-statement", statementCronInterval, runWeeklyStatementScan, StartWeeklyStatementCron},
		{"reconciliation", reconciliationInterval, runReconciliationScan, StartReconciliationCron},
		{"fssai-reminder", fssaiReminderInterval, runFSSAIReminderScan, StartFSSAIReminderCron},
		{"availability-resume", availabilityResumeInterval, runAvailabilityResumeScan, StartAvailabilityResumeCron},
		// Schedule-driven auto open/close: flip AcceptingOrders to match the chef's
		// operating hours for opted-in kitchens (AutoScheduleEnabled).
		{"kitchen-schedule", kitchenScheduleInterval, runKitchenScheduleScan, StartKitchenScheduleCron},
		{"audit-retention", auditRetentionInterval, runAuditRetentionScan, StartAuditRetentionCron},
		{"meal-plan-sweep", mealPlanSweepInterval, runMealPlanSweep, StartMealPlanCron},
		{"meal-plan-fulfillment", mealPlanFulfillmentInterval, runMealPlanFulfillment, StartMealPlanFulfillmentCron},
		{"group-order-sweep", groupOrderSweepInterval, runGroupOrderSweep, StartGroupOrderCron},
		{"winback-scan", winbackScanInterval, runWinbackScan, StartWinbackCron},
		{"meal-sub-orders", mealSubOrderScanInterval, runMealSubscriptionDailyOrders, StartMealSubscriptionOrderCron},
		{"campaign-dispatch", campaignDispatchInterval, runCampaignDispatch, StartCampaignCron},
		{"stale-order", staleOrderInterval, runStaleOrderScan, StartStaleOrderCron},
		// #694 — void + refund paid orders the chef never accepted before their
		// kitchen closed. Registered here so it runs under Temporal Schedules where
		// they drive the crons, and as an in-process ticker where they do not: the
		// money hole must not depend on which mode we are in.
		{"unaccepted-order", unacceptedOrderInterval, runUnacceptedOrderScan, StartUnacceptedOrderCron},
		// #694 — the pre-close nudge: remind a chef of an order they have not
		// accepted, before the void sweep cancels it. Same durable/ticker duality.
		{"accept-reminder", acceptReminderInterval, runAcceptReminderScan, StartAcceptReminderCron},
		{"payout-auto-confirm", payoutAutoConfirmInterval, runPayoutAutoConfirmScan, StartPayoutAutoConfirmCron},
		{"payout-reconcile", payoutReconcileInterval, runPayoutReconcileScan, StartPayoutReconcileCron},
		{"cancellation-sweep", cancellationSweepInterval, runCancellationSweep, StartCancellationCron},
		// #741 — release matured, unblocked order payouts. Gated by
		// payout.sweep_enabled, which ships off.
		{"payout-release", payoutReleaseInterval, runPayoutReleaseSweep, StartPayoutReleaseCron},
	}
}

// CronJobActivity runs one registered scheduled job by name. Registered on the
// cron worker (cmd/worker) so Temporal Schedules can trigger it.
func CronJobActivity(ctx context.Context, name string) error {
	for _, j := range cronJobs() {
		if j.name == name {
			j.run(ctx)
			return nil
		}
	}
	return fmt.Errorf("cron: unknown job %q", name)
}

// CronJobWorkflow is the thin workflow a Schedule triggers; it just runs the job
// activity (with retry until its timeout).
func CronJobWorkflow(ctx workflow.Context, name string) error {
	ctx = apitemporal.Activities(ctx, 30*time.Minute)
	return workflow.ExecuteActivity(ctx, CronJobActivity, name).Get(ctx, nil)
}

// RegisterCronWorker wires the cron workflow + activity onto a worker spec.
func RegisterCronWorker() apitemporal.WorkerSpec {
	return apitemporal.Queue(apitemporal.TaskQueueCron).
		Workflows(CronJobWorkflow).
		Activities(CronJobActivity)
}

// StartCronJobs starts the platform's scheduled jobs. When Temporal is enabled
// it creates idempotent Schedules; otherwise it starts the legacy in-process
// tickers so behaviour is unchanged until the cluster (#119) exists.
func StartCronJobs(ctx context.Context) {
	if temporalRT == nil {
		log.Println("Cron: Temporal disabled — using in-process tickers")
		startTickers(ctx)
		return
	}
	if err := ensureSchedules(ctx); err != nil {
		log.Printf("Cron: Temporal schedule setup failed (%v) — falling back to in-process tickers", err)
		startTickers(ctx)
		return
	}
	log.Println("Cron: running via Temporal Schedules (exactly-once)")
}

func startTickers(ctx context.Context) {
	for _, j := range cronJobs() {
		j.ticker(ctx)
	}
}

// ensureSchedules creates a Temporal Schedule per job (idempotent: an existing
// schedule is left as-is). Schedules persist on the server across restarts.
func ensureSchedules(ctx context.Context) error {
	sc := temporalRT.Client().ScheduleClient()
	for _, j := range cronJobs() {
		_, err := sc.Create(ctx, client.ScheduleOptions{
			ID: "homechef-cron-" + j.name,
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{{Every: j.interval}},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        "homechef:cron:" + j.name,
				Workflow:  CronJobWorkflow,
				Args:      []interface{}{j.name},
				TaskQueue: apitemporal.TaskQueueCron,
			},
		})
		// Create returns an error if the schedule already exists — expected on
		// every boot after the first, so log and continue rather than fail.
		if err != nil {
			log.Printf("Cron: schedule %q already present or not created: %v", j.name, err)
		}
	}
	return nil
}
