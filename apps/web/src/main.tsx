import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './app/App';
import { initAnalytics } from './lib/analytics';
import './styles/globals.css';

const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Root element not found');
}

// Register service worker for PWA support
if ('serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', async () => {
    try {
      await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
        // Bypass the HTTP cache when checking for a new worker. Without it a
        // cached sw.js pins users to a stale app shell across deploys.
        updateViaCache: 'none',
      });
    } catch (error) {
      // SW registration failure is non-fatal; degrade silently to no-offline mode.
      // Surface to error monitoring if you wire one up.
      void error;
    }
  });
}

initAnalytics();

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>
);
