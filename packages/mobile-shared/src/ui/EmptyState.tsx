import type { ReactNode } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { theme } from '../theme/tokens';
import { Button } from './Button';

interface EmptyStateProps {
  /** Optional graphic — an icon, illustration, or any element ≤ 96pt tall.
   *  The component reserves space and centres the element. */
  icon?: ReactNode;
  /** Single short line, sentence case. Avoid "No data!" exclamation. */
  title: string;
  /** Optional supporting copy. Keep to 1-2 lines. */
  body?: string;
  /** Optional primary action. The CTA is always persimmon — never
   *  competes with the title. */
  ctaLabel?: string;
  onCtaPress?: () => void;
  /** Optional secondary action (e.g. "Learn more"). Renders as a ghost
   *  button beneath the primary CTA. */
  secondaryLabel?: string;
  onSecondaryPress?: () => void;
}

/**
 * <EmptyState> — the canonical "no data yet" / "no results" / "you're
 * caught up" surface.
 *
 * Centred composition: icon → title → body → CTA. The icon is gently
 * tinted (mist surface, ink glyph). No big stock illustrations, no
 * "We searched everywhere..." copy. One sentence, one action.
 */
export function EmptyState({
  icon,
  title,
  body,
  ctaLabel,
  onCtaPress,
  secondaryLabel,
  onSecondaryPress,
}: EmptyStateProps) {
  return (
    <View style={styles.root}>
      {icon ? <View style={styles.iconWrap}>{icon}</View> : null}
      <Text style={styles.title}>{title}</Text>
      {body ? <Text style={styles.body}>{body}</Text> : null}
      {ctaLabel && onCtaPress ? (
        <View style={styles.cta}>
          <Button label={ctaLabel} onPress={onCtaPress} fullWidth={false} />
        </View>
      ) : null}
      {secondaryLabel && onSecondaryPress ? (
        <View style={styles.secondary}>
          <Button
            label={secondaryLabel}
            variant="ghost"
            onPress={onSecondaryPress}
            fullWidth={false}
          />
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[12],
    gap: theme.spacing[2],
  },
  iconWrap: {
    width: 64,
    height: 64,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.mist.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: theme.spacing[3],
  },
  title: {
    fontFamily: 'Geist',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
  },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight: theme.typography.size.body.size * theme.typography.size.body.lineHeight,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    maxWidth: 320,
  },
  cta: { marginTop: theme.spacing[4] },
  secondary: { marginTop: theme.spacing[1] },
});
