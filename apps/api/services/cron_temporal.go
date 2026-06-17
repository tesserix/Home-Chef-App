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
		{"audit-retention", auditRetentionInterval, runAuditRetentionScan, StartAuditRetentionCron},
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
