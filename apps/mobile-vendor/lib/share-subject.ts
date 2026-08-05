// share-subject.ts — what the Promote screen is currently sharing.
//
// The screen used to render every post as its own always-visible radio row and
// every network as its own always-visible button, so a chef with five posts read
// eleven rows before finding the share they wanted. The picker collapses to one
// row; this holds the list behind it.

/** The chef's public page, as opposed to one of their posts. */
export const KITCHEN_SUBJECT = 'kitchen';

export interface ShareSubjectOption {
  id: string;
  title: string;
}

interface ArticleLike {
  id: string;
  title: string;
  status: string;
}

/** The kitchen first, then each PUBLISHED post — a draft has no page to send anyone to. */
export function shareSubjectOptions(
  articles: ArticleLike[],
  businessName: string,
): ShareSubjectOption[] {
  return [
    { id: KITCHEN_SUBJECT, title: businessName || 'Your kitchen' },
    ...articles
      .filter((a) => a.status === 'published')
      .map((a) => ({ id: a.id, title: a.title })),
  ];
}

/** The selected option, falling back to the kitchen so a post deleted elsewhere
 *  can never strand the picker on an id that no longer resolves. */
export function resolveShareSubject(
  options: ShareSubjectOption[],
  subject: string | undefined,
): ShareSubjectOption | undefined {
  return options.find((o) => o.id === subject) ?? options[0];
}
