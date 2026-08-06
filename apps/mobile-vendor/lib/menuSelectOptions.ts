// Maps the menu form's dropdown labels to the values the API stores. A dropdown
// works in names; a menu item is saved with a category id and a number of minutes.

export interface CategoryOption {
  id: string;
  name: string;
}

export function categoryNameForId(categories: CategoryOption[], id: string): string {
  return categories.find((c) => c.id === id)?.name ?? '';
}

export function categoryIdForName(categories: CategoryOption[], name: string): string {
  return categories.find((c) => c.name === name)?.id ?? '';
}

export function prepTimeLabel(minutes: number): string {
  return `${minutes} min`;
}

export function prepTimeForLabel(label: string): number | null {
  const minutes = Number.parseInt(label, 10);
  return Number.isNaN(minutes) ? null : minutes;
}
