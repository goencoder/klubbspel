/**
 * Session Manager Module
 * 
 * Provides a centralized way to handle session expiration without circular dependencies.
 * This module acts as a bridge between the API client and auth store.
 */

type SessionExpiredHandler = () => void

let sessionExpiredHandler: SessionExpiredHandler | null = null

/**
 * Register a callback to be invoked when a session expires.
 * This should be called once during app initialization by the auth store.
 */
export function registerSessionExpiredHandler(handler: SessionExpiredHandler) {
  sessionExpiredHandler = handler
}

/**
 * Trigger the session expired handler.
 * This is called by the API client when it receives an INVALID_OR_EXPIRED_TOKEN error.
 */
export function handleSessionExpired() {
  if (sessionExpiredHandler) {
    sessionExpiredHandler()
  } else {
    console.warn('Session expired but no handler registered')
  }
}
