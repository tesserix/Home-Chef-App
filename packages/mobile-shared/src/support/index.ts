// Barrel for the shared mobile support-chat module (WS→poll; no SSE).
export * from "./types";
export * from "./events";
export * from "./outbox";
export * from "./client";
export * from "./sse";
export * from "./useSupportChat";
export * from "./SupportChatView";
export * from "./SupportChatHistory";
export { secureStoreKV } from "./storage";
