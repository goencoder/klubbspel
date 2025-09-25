// Error handling utilities for consistent error management across the app

import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { isRequestCancelled } from '@/services/api'
import type { ApiError } from '@/types/api'

/**
 * Safely handles API errors by checking if they're cancelled requests first
 * and only showing toast messages for actual errors
 */
export function safeApiCall<T>(
  apiCall: () => Promise<T>,
  errorMessage?: string
): Promise<T | null> {
  return apiCall().catch((error: unknown) => {
    if (isRequestCancelled(error)) {
      // Silently ignore cancelled requests
      return null
    }
    
    const apiError = error as ApiError
    if (errorMessage) {
      toast.error(errorMessage)
    } else {
      toast.error(apiError.message || 'An unexpected error occurred')
    }
    
    return null
  })
}

/**
 * Hook for handling API errors with translation support
 */
export function useErrorHandler() {
  const { t } = useTranslation()
  
  const handleError = (error: unknown, customMessage?: string) => {
    if (isRequestCancelled(error)) {
      // Silently ignore cancelled requests
      return
    }
    
    const apiError = error as ApiError
    const message = customMessage || apiError.message || t('errors.generic')
    toast.error(message)
  }
  
  return { handleError }
}

/**
 * Wrapper for async functions that automatically handles cancelled requests
 */
export async function withCancellationHandling<T>(
  asyncFn: () => Promise<T>,
  onError?: (error: ApiError) => void
): Promise<T | undefined> {
  try {
    return await asyncFn()
  } catch (error) {
    if (isRequestCancelled(error)) {
      // Silently ignore cancelled requests
      return undefined
    }
    
    const apiError = error as ApiError
    onError?.(apiError)
    throw error
  }
}