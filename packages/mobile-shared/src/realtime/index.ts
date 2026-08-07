// Mobile realtime is SSE only: React Native's WebSocket cannot complete a TLS
// handshake against our edge, so it never issues a request at all (#982).
export { openEventStream, type EventStreamHandle } from './event-stream';
export { streamRetryPlan, type StreamRetryPlan } from './stream-retry';
