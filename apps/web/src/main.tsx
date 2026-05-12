import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './app/App';
import './styles/globals.css';

const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Root element not found');
}

// Register service worker for PWA support
if ('serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', async () => {
    try {
      await navigator.serviceWorker.register('/sw.js', { scope: '/' });
    } catch (error) {
      // SW registration failure is non-fatal; degrade silently to no-offline mode.
      // Surface to error monitoring if you wire one up.
      void error;
    }
  });
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>
);
