import { isDeclinedDayStatus } from './meal-plan';

// Approval-banner copy (#1036). The banner used to be a fixed string — "Your
// chef revised this plan / They can cook 5 of 5 days" — which contradicted
// itself the moment the chef accepted everything, which is the common case.
// The wording has to be derived from what the chef actually did, so it lives
// here as a pure function rather than as conditionals inline in the JSX: the
// plan-detail screen, and any other surface that grows this banner later, then
// share one set of strings that a test can pin down.
//
// Terminology (#1040): `plan.days` holds booked MEALS, not calendar days — a
// plan can book lunch AND dinner on the same date — so this copy says
// "meal"/"meals" throughout. Never "days".

export interface MealPlanApprovalCopy {
  /** Banner heading — what the chef did. */
  title: string;
  /** Banner body — the counts plus what approve/reject will do. */
  body: string;
  /** Meals the chef will cook, i.e. what approving buys. */
  acceptedCount: number;
  /** Meals the chef could not take, i.e. what drops on approval. */
  declinedCount: number;
}

/**
 * mealPlanApprovalCopy chooses the approval-banner wording for a plan awaiting
 * the customer's decision (`awaiting_customer` / `chef_modified`).
 *
 * `isDeclinedDayStatus` (declined/skipped/cancelled/refunded/failed) is the one
 * rule for "this meal will not be served", shared with the pricing subset
 * (#1039) and the struck-through rows, so the banner's counts can never
 * disagree with the money or the list beneath it.
 */
export function mealPlanApprovalCopy(
  days: { status: string }[] = [],
): MealPlanApprovalCopy {
  const total = days.length;
  const acceptedCount = days.filter((d) => !isDeclinedDayStatus(d.status)).length;
  const declinedCount = total - acceptedCount;

  // No days at all. Shouldn't happen for a real plan, but the screen renders
  // from cache before the days land, and a heading that asserts something false
  // is worse than a neutral one.
  if (total === 0) {
    return {
      title: 'Your approval needed',
      body: 'Your chef has responded to this plan. Approve to confirm, or reject to cancel it.',
      acceptedCount,
      declinedCount,
    };
  }

  // The chef took the whole plan. Saying "revised" here is simply untrue, and
  // it primes the customer to hunt for a change that does not exist (#1036).
  if (declinedCount === 0) {
    return {
      title: 'Your chef accepted your plan',
      body:
        total === 1
          ? 'They can cook the meal you asked for. Approve to confirm it, or reject to cancel the plan.'
          : `They can cook all ${total} meals. Approve to confirm them, or reject to cancel the whole plan.`,
      acceptedCount,
      declinedCount,
    };
  }

  // The chef took nothing. "Revised" is wrong here too — there is no trimmed
  // plan to approve, so the copy points at the only useful action.
  if (acceptedCount === 0) {
    return {
      title: "Your chef can't cook this plan",
      body:
        total === 1
          ? "They couldn't take the meal you asked for. Reject to cancel the plan — you can book another chef any time."
          : `They couldn't take any of the ${total} meals you asked for. Reject to cancel the plan — you can book another chef any time.`,
      acceptedCount,
      declinedCount,
    };
  }

  // Genuinely trimmed — the original wording, now stating how many dropped so
  // the customer can see the gap without counting the struck-through rows.
  return {
    title: 'Your chef revised this plan',
    body: `They can cook ${acceptedCount} of ${total} meals — ${
      declinedCount === 1 ? '1 meal' : `${declinedCount} meals`
    } dropped. Approve to confirm the rest, or reject to cancel the whole plan.`,
    acceptedCount,
    declinedCount,
  };
}
