import { describe, it, expect } from '@jest/globals';

import { KITCHEN_SUBJECT, shareSubjectOptions, resolveShareSubject } from './share-subject';

const posts = [
  { id: 'a1', title: 'Sunday biryani, from scratch', status: 'published' },
  { id: 'a2', title: 'Draft: monsoon menu', status: 'draft' },
  { id: 'a3', title: 'Why I salt my dal last', status: 'published' },
];

describe('shareSubjectOptions', () => {
  it('leads with the kitchen, then every published post', () => {
    expect(shareSubjectOptions(posts, 'Dum Alooo Kitchen')).toEqual([
      { id: KITCHEN_SUBJECT, title: 'Dum Alooo Kitchen' },
      { id: 'a1', title: 'Sunday biryani, from scratch' },
      { id: 'a3', title: 'Why I salt my dal last' },
    ]);
  });

  it('never offers a draft — readers would land on a page that is not there', () => {
    const ids = shareSubjectOptions(posts, 'K').map((o) => o.id);
    expect(ids).not.toContain('a2');
  });

  it('falls back to a generic kitchen label before the profile loads', () => {
    expect(shareSubjectOptions([], '')[0]).toEqual({
      id: KITCHEN_SUBJECT,
      title: 'Your kitchen',
    });
  });

  it('offers the kitchen alone when nothing is published', () => {
    expect(shareSubjectOptions([], 'K')).toHaveLength(1);
  });
});

describe('resolveShareSubject', () => {
  const options = shareSubjectOptions(posts, 'K');

  it('returns the chosen option', () => {
    expect(resolveShareSubject(options, 'a3')?.title).toBe('Why I salt my dal last');
  });

  it('falls back to the kitchen when the chosen post is gone', () => {
    // A post deleted or unpublished in another tab must not strand the picker on
    // an id that no longer resolves.
    expect(resolveShareSubject(options, 'deleted')?.id).toBe(KITCHEN_SUBJECT);
  });

  it('falls back to the kitchen for an unset subject', () => {
    expect(resolveShareSubject(options, undefined)?.id).toBe(KITCHEN_SUBJECT);
  });

  it('returns undefined when there is nothing to share at all', () => {
    expect(resolveShareSubject([], 'a1')).toBeUndefined();
  });
});
