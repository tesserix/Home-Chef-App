import { useState, useEffect, useCallback } from 'react';

interface ServiceWorkerState {
  isSupported: boolean;
  isRegistered: boolean;
  isUpdateAvailable: boolean;
  isOffline: boolean;
  registration: ServiceWorkerRegistration | null;
}

interface UseServiceWorkerReturn extends ServiceWorkerState {
  update: () => Promise<void>;
  unregister: () => Promise<boolean>;
}

export function useServiceWorker(): UseServiceWorkerReturn {
  const [state, setState] = useState<ServiceWorkerState>({
    isSupported: false,
    isRegistered: false,
    isUpdateAvailable: false,
    isOffline: false,
    registration: null,
  });

  useEffect(() => {
    const isSupported = 'serviceWorker' in navigator;
    setState((prev) => ({ ...prev, isSupported }));

    if (!isSupported) return;

    // Register service worker
    const registerSW = async () => {
      try {
        const registration = await navigator.serviceWorker.register('/sw.js', {
          scope: '/',
        });

        setState((prev) => ({
          ...prev,
          isRegistered: true,
          registration,
        }));

        // Check for updates
        registration.addEventListener('updatefound', () => {
          const newWorker = registration.installing;
          if (newWorker) {
            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                setState((prev) => ({ ...prev, isUpdateAvailable: true }));
              }
            });
          }
        });

        // Check for updates periodically
        setInterval(() => {
          registration.update();
        }, 60 * 60 * 1000); // Every hour
      } catch (error) {
        console.error('Service worker registration failed:', error);
      }
    };

    registerSW();

    // Listen for online/offline status
    const handleOnline = () => setState((prev) => ({ ...prev, isOffline: false }));
    const handleOffline = () => setState((prev) => ({ ...prev, isOffline: true }));

    setState((prev) => ({ ...prev, isOffline: !navigator.onLine }));
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  const update = useCallback(async () => {
    if (state.registration?.waiting) {
      state.registration.waiting.postMessage({ type: 'SKIP_WAITING' });
      window.location.reload();
    }
  }, [state.registration]);

  const unregister = useCallback(async () => {
    if (state.registration) {
      return state.registration.unregister();
    }
    return false;
  }, [state.registration]);

  return {
    ...state,
    update,
    unregister,
  };
}

/**
 * Hook for requesting push notification permissions
 */
export function usePushNotifications() {
  const [permission, setPermission] = useState<NotificationPermission>('default');
  const [subscription, setSubscription] = useState<PushSubscription | null>(null);

  useEffect(() => {
    if ('Notification' in window) {
      setPermission(Notification.permission);
    }
  }, []);

  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!('Notification' in window)) {
      console.warn('Notifications not supported');
      return false;
    }

    const result = await Notification.requestPermission();
    setPermission(result);
    return result === 'granted';
  }, []);

  const subscribe = useCallback(async (vapidPublicKey: string): Promise<PushSubscription | null> => {
    try {
      const registration = await navigator.serviceWorker.ready;
      const existingSubscription = await registration.pushManager.getSubscription();

      if (existingSubscription) {
        setSubscription(existingSubscription);
        return existingSubscription;
      }

      const newSubscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
      });

      setSubscription(newSubscription);
      return newSubscription;
    } catch (error) {
      console.error('Push subscription failed:', error);
      return null;
    }
  }, []);

  const unsubscribe = useCallback(async (): Promise<boolean> => {
    if (subscription) {
      const result = await subscription.unsubscribe();
      if (result) {
        setSubscription(null);
      }
      return result;
    }
    return false;
  }, [subscription]);

  return {
    permission,
    subscription,
    isSupported: 'PushManager' in window,
    isGranted: permission === 'granted',
    isDenied: permission === 'denied',
    requestPermission,
    subscribe,
    unsubscribe,
  };
}

// Helper to convert VAPID key
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/**
 * Hook for background sync
 */
export function useBackgroundSync() {
  const [isSupported, setIsSupported] = useState(false);

  useEffect(() => {
    setIsSupported('serviceWorker' in navigator && 'SyncManager' in window);
  }, []);

  const requestSync = useCallback(async (tag: string): Promise<boolean> => {
    if (!isSupported) return false;

    try {
      const registration = await navigator.serviceWorker.ready;
      // @ts-expect-error - SyncManager types not fully available
      await registration.sync.register(tag);
      return true;
    } catch (error) {
      console.error('Background sync registration failed:', error);
      return false;
    }
  }, [isSupported]);

  return {
    isSupported,
    requestSync,
  };
}
