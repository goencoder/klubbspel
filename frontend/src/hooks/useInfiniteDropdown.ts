import { useState, useCallback, useRef, useEffect } from 'react'
import { useDebounce } from './useDebounce'
import { isRequestCancelled } from '@/services/api'

export interface UseInfiniteDropdownOptions<T> {
  loadItems: (query: string, pageToken?: string) => Promise<{
    items: T[]
    hasNextPage: boolean
    endCursor?: string
  }>
  debounceMs?: number
  pageSize?: number
  initialQuery?: string
}

export interface UseInfiniteDropdownResult<T> {
  // State
  items: T[]
  loading: boolean
  loadingMore: boolean
  hasNextPage: boolean
  query: string
  open: boolean
  
  // Actions
  setQuery: (query: string) => void
  setOpen: (open: boolean) => void
  loadMore: () => void
  refresh: () => void
  
  // Utilities
  isEmpty: boolean
  isFirstLoad: boolean
}

export function useInfiniteDropdown<T>({
  loadItems,
  debounceMs = 300,
  initialQuery = ''
}: UseInfiniteDropdownOptions<T>): UseInfiniteDropdownResult<T> {
  const [items, setItems] = useState<T[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasNextPage, setHasNextPage] = useState(false)
  const [query, setQuery] = useState(initialQuery)
  const [open, setOpen] = useState(false)
  const [nextPageToken, setNextPageToken] = useState<string | undefined>()
  
  const debouncedQuery = useDebounce(query, debounceMs)
  const abortControllerRef = useRef<AbortController | null>(null)
  const isFirstLoadRef = useRef(true)
  
  const load = useCallback(async (searchQuery: string, isLoadMore = false) => {
    // Cancel previous request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
    
    const controller = new AbortController()
    abortControllerRef.current = controller
    
    try {
      if (isLoadMore) {
        setLoadingMore(true)
      } else {
        setLoading(true)
        isFirstLoadRef.current = false
      }
      
      const response = await loadItems(
        searchQuery,
        isLoadMore ? nextPageToken : undefined
      )
      
      // Ignore if request was aborted
      if (controller.signal.aborted) {
        return
      }
      
      if (isLoadMore) {
        setItems(prev => [...prev, ...response.items])
      } else {
        setItems(response.items)
      }
      
      setHasNextPage(response.hasNextPage)
      setNextPageToken(response.endCursor)
    } catch (error) {
      // Ignore abort errors and cancelled requests
      if (error instanceof Error && error.name === 'AbortError' || isRequestCancelled(error)) {
        return
      }
      
      // Reset state on error
      if (!isLoadMore) {
        setItems([])
        setHasNextPage(false)
        setNextPageToken(undefined)
      }
      
      throw error
    } finally {
      setLoading(false)
      setLoadingMore(false)
      abortControllerRef.current = null
    }
  }, [loadItems, nextPageToken])
  
  const loadMore = useCallback(() => {
    if (hasNextPage && !loadingMore && !loading) {
      load(debouncedQuery, true)
    }
  }, [hasNextPage, loadingMore, loading, load, debouncedQuery])
  
  const refresh = useCallback(() => {
    setNextPageToken(undefined)
    load(debouncedQuery, false)
  }, [load, debouncedQuery])
  
  // Load data when query changes
  useEffect(() => {
    if (open || debouncedQuery) {
      setNextPageToken(undefined)
      load(debouncedQuery, false)
    }
  }, [debouncedQuery, open, load])
  
  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])
  
  return {
    // State
    items,
    loading,
    loadingMore,
    hasNextPage,
    query,
    open,
    
    // Actions
    setQuery,
    setOpen,
    loadMore,
    refresh,
    
    // Utilities
    isEmpty: items.length === 0,
    isFirstLoad: isFirstLoadRef.current
  }
}