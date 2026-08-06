// Blocks of the Profile form a deep link can scroll to. Bakery is an alias for
// the kitchen block, because "sell cakes and bakes" is what a chef goes looking
// for and "KITCHEN" is what the block is called (#1065).
export type ProfileSection = 'kitchen';

const SECTION_ALIASES: Record<string, ProfileSection> = {
  kitchen: 'kitchen',
  bakery: 'kitchen',
};

/** Maps a `?section=` query value onto the block to scroll to, or null. */
export function focusedProfileSection(
  param: string | string[] | undefined,
): ProfileSection | null {
  const raw = Array.isArray(param) ? param[0] : param;
  if (!raw) return null;
  return SECTION_ALIASES[raw.trim().toLowerCase()] ?? null;
}
