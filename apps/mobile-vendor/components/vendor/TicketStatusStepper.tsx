import { StyleSheet, Text, View } from 'react-native';
import { Check } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import type { TicketStatus } from '../../hooks/useSupport';

// Read-only 4-stage progress stepper. The six backend statuses collapse
// onto four user-facing stages — `waiting_*` reads as "In progress" since
// the chef can't act on the distinction. Purely informational: the chef
// never triggers a transition from here (only the support team does, plus
// the chef's own Close action elsewhere on the screen).
type Stage = 'open' | 'in_progress' | 'resolved' | 'closed';

const STAGES: { stage: Stage; label: string }[] = [
  { stage: 'open', label: 'Opened' },
  { stage: 'in_progress', label: 'In progress' },
  { stage: 'resolved', label: 'Resolved' },
  { stage: 'closed', label: 'Closed' },
];

function stageIndex(status: TicketStatus): number {
  switch (status) {
    case 'open':
      return 0;
    case 'in_progress':
    case 'waiting_on_customer':
    case 'waiting_on_chef':
      return 1;
    case 'resolved':
      return 2;
    case 'closed':
      return 3;
    default:
      return 0;
  }
}

// The "Resolved" stage carries the success-green accent once reached; every
// other completed/current stage is ink.
function fillColor(stage: Stage): string {
  return stage === 'resolved'
    ? theme.colors.success.DEFAULT
    : theme.colors.ink.DEFAULT;
}

export function TicketStatusStepper({ status }: { status: TicketStatus }) {
  const active = stageIndex(status);

  return (
    <View style={styles.row} accessibilityRole="progressbar">
      {STAGES.map((s, idx) => {
        const isPast = idx < active;
        const isActive = idx === active;
        const done = isPast || isActive;
        const fill = fillColor(s.stage);

        return (
          <View key={s.stage} style={styles.stepWrap}>
            <View style={styles.circleRow}>
              {/* left half-connector */}
              <View
                style={[
                  styles.connector,
                  idx === 0 && styles.connectorHidden,
                  {
                    backgroundColor:
                      isPast || isActive ? fill : theme.colors.mist.DEFAULT,
                  },
                ]}
              />
              <View
                style={[
                  styles.circle,
                  done
                    ? { backgroundColor: fill, borderColor: fill }
                    : {
                        backgroundColor: theme.colors.paper,
                        borderColor: theme.colors.mist.strong,
                      },
                ]}
              >
                {isPast ? (
                  <Check
                    size={14}
                    color={theme.colors.paper}
                    strokeWidth={2.5}
                  />
                ) : (
                  <Text
                    style={[
                      styles.circleNum,
                      {
                        color: done
                          ? theme.colors.paper
                          : theme.colors.ink.muted,
                      },
                    ]}
                  >
                    {idx + 1}
                  </Text>
                )}
              </View>
              {/* right half-connector */}
              <View
                style={[
                  styles.connector,
                  idx === STAGES.length - 1 && styles.connectorHidden,
                  {
                    backgroundColor:
                      idx < active
                        ? fillColor(STAGES[idx + 1]!.stage)
                        : theme.colors.mist.DEFAULT,
                  },
                ]}
              />
            </View>
            <Text
              style={[
                styles.label,
                isActive
                  ? styles.labelActive
                  : isPast
                    ? styles.labelPast
                    : styles.labelUpcoming,
              ]}
              numberOfLines={1}
            >
              {s.label}
            </Text>
          </View>
        );
      })}
    </View>
  );
}

const CIRCLE = 28;

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  stepWrap: {
    flex: 1,
    alignItems: 'center',
  },
  circleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    width: '100%',
  },
  connector: {
    flex: 1,
    height: 2,
  },
  connectorHidden: {
    backgroundColor: 'transparent',
  },
  circle: {
    width: CIRCLE,
    height: CIRCLE,
    borderRadius: CIRCLE / 2,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  circleNum: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
  },
  label: {
    marginTop: 6,
    fontSize: theme.typography.size.caption.size,
    textAlign: 'center',
  },
  labelActive: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
  },
  labelPast: {
    fontFamily: 'Inter',
    color: theme.colors.ink.soft,
  },
  labelUpcoming: {
    fontFamily: 'Inter',
    color: theme.colors.ink.muted,
  },
});
