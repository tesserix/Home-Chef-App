// Leaving a screen that may have been opened directly (push notification,
// deep link) — router.back() alone fails there and strands the chef.

interface Navigator {
  canGoBack: () => boolean;
  back: () => void;
  replace: (href: never) => void;
}

/** Go back if there is a stack, otherwise land on `fallback`. */
export function leaveTo<R extends Navigator>(router: R, fallback: string): void {
  if (router.canGoBack()) {
    router.back();
    return;
  }
  (router.replace as (href: string) => void)(fallback);
}
