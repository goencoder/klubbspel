import apiClient from '@/services/api'
import type { AuthContext, CurrentUser } from '@/types/membership'
import { registerSessionExpiredHandler } from '@/lib/sessionManager'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  user: CurrentUser | null
  selectedClubId: string | null
  isLoading: boolean
  error: string | null
  sessionExpired: boolean
}

interface AuthActions {
  // Authentication actions
  sendMagicLink: (email: string, returnUrl?: string) => Promise<void>
  validateToken: (token: string) => Promise<void>
  logout: () => void

  // Session management
  handleSessionExpired: () => void
  dismissSessionExpired: () => void

  // Club selection
  selectClub: (clubId: string | null) => void

  // User management
  refreshUserMemberships: () => Promise<void>
  refreshUser: () => Promise<void>

  // Helper functions
  isAuthenticated: () => boolean
  isPlatformOwner: () => boolean
  isClubAdmin: (clubId: string) => boolean
  isClubMember: (clubId: string) => boolean

  // State management
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  clearError: () => void
}

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      selectedClubId: null,
      isLoading: false,
      error: null,
      sessionExpired: false,

      // Authentication actions
      sendMagicLink: async (email: string, returnUrl?: string) => {
        set({ isLoading: true, error: null })
        try {
          await apiClient.sendMagicLink({ email, returnUrl })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to send magic link'
          set({ error: errorMessage })
          throw error
        } finally {
          set({ isLoading: false })
        }
      },

      validateToken: async (token: string) => {
        set({ isLoading: true, error: null })
        try {
          const response = await apiClient.validateToken({ token })

          if (response.apiToken && response.user?.email && response.user?.id) {
            const user: CurrentUser = {
              email: response.user.email,
              displayName: `${response.user.firstName || ''} ${response.user.lastName || ''}`.trim() || response.user.email,
              playerId: response.user.id, // Use ID directly from auth response
              apiToken: response.apiToken,
              memberships: response.user.clubMemberships || [],
              isPlatformOwner: response.user.isPlatformOwner || false
            }

            set({ user, selectedClubId: user.memberships[0]?.clubId || null })
          } else {
            throw new Error('Invalid token response - missing required fields')
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Token validation failed'
          set({ error: errorMessage, user: null })
          throw error
        } finally {
          set({ isLoading: false })
        }
      },

      logout: () => {
        set({
          user: null,
          selectedClubId: null,
          isLoading: false,
          error: null,
          sessionExpired: false
        })
      },

      // Session management
      handleSessionExpired: () => {
        set({
          user: null,
          selectedClubId: null,
          sessionExpired: true,
          error: null
        })
      },

      dismissSessionExpired: () => {
        set({ sessionExpired: false })
      },

      // Club selection
      selectClub: (clubId: string | null) => {
        set({ selectedClubId: clubId })
      },

      // User management
      refreshUserMemberships: async () => {
        const { user } = get()
        if (!user) {return}

        try {
          // Use getCurrentUser to get fresh user data including memberships
          const response = await apiClient.getCurrentUser()
          
          if (response.user?.id) {
            set({
              user: {
                ...user,
                playerId: response.user.id, // Ensure player ID is kept consistent
                memberships: response.user.clubMemberships || [],
                isPlatformOwner: response.user.isPlatformOwner || false
              }
            })
          }
        } catch (_error) {
          // Don't set error state for background refresh
        }
      },

      // Refresh current user profile data
      refreshUser: async () => {
        const { user } = get()
        if (!user) {return}

        try {
          const response = await apiClient.getCurrentUser()
          
          if (response.user?.id) {
            const updatedUser: CurrentUser = {
              email: response.user.email,
              displayName: `${response.user.firstName || ''} ${response.user.lastName || ''}`.trim() || response.user.email,
              playerId: response.user.id, // Use ID directly from getCurrentUser response
              apiToken: user.apiToken, // Keep existing token
              memberships: response.user.clubMemberships || [],
              isPlatformOwner: response.user.isPlatformOwner || false
            }

            set({ user: updatedUser })
          }
        } catch (_error) {
          // Don't set error state for background refresh
        }
      },

      // Helper functions
      isAuthenticated: () => {
        const { user } = get()
        try {
          return user !== null && user?.apiToken !== ''
        } catch {
          get().logout()
          return false
        }
      },

      isPlatformOwner: () => {
        const { user } = get()
        try {
          return user?.isPlatformOwner || false
        } catch {
          get().logout()
          return false
        }
      },

      isClubAdmin: (clubId: string) => {
        const { user } = get()
        try {
          if (!user || !user.memberships) {
            return false
          }

          const membership = user.memberships.find(m => m?.clubId === clubId)
          return membership?.role === 'MEMBERSHIP_ROLE_ADMIN'
        } catch {
          get().logout()
          return false
        }
      },

      isClubMember: (clubId: string) => {
        const { user } = get()
        try {
          if (!user || !user.memberships) {
            return false
          }

          const membership = user.memberships.find(m => m?.clubId === clubId)
          return membership !== undefined
        } catch {
          get().logout()
          return false
        }
      },

      // State management
      setLoading: (loading: boolean) => set({ isLoading: loading }),
      setError: (error: string | null) => set({ error }),
      clearError: () => set({ error: null })
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        selectedClubId: state.selectedClubId
      })
    }
  )
)

// Helper hook to get auth context
export const useAuthContext = (): AuthContext => {
  const {
    user,
    selectedClubId,
    isAuthenticated,
    isPlatformOwner,
    isClubAdmin,
    isClubMember
  } = useAuthStore()

  return {
    user,
    selectedClubId,
    isAuthenticated: isAuthenticated(),
    isPlatformOwner: isPlatformOwner(),
    isClubAdmin,
    isClubMember
  }
}

// Register the session expired handler with the session manager
// This breaks the circular dependency between api.ts and auth store
registerSessionExpiredHandler(() => {
  useAuthStore.getState().handleSessionExpired()
})

// API header injection for authenticated requests
export const useApiHeaders = () => {
  const { user } = useAuthStore()

  return () => {
    const headers: Record<string, string> = {}

    if (user?.apiToken) {
      headers['Authorization'] = `Bearer ${user.apiToken}`
    }

    return headers
  }
}
