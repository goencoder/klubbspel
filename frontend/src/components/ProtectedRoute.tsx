import { useAuthStore } from '@/store/auth'
import React from 'react'
import { Navigate, useLocation } from 'react-router-dom'

interface ProtectedRouteProps {
  children: React.ReactNode
  requireClubMembership?: string // Club ID if membership is required
  requireClubAdmin?: string // Club ID if admin role is required
  requirePlatformOwner?: boolean
}

export function ProtectedRoute({
  children,
  requireClubMembership,
  requireClubAdmin,
  requirePlatformOwner = false
}: ProtectedRouteProps) {
  const location = useLocation()
  const { isAuthenticated, isClubMember, isClubAdmin, isPlatformOwner } = useAuthStore()

  // Check if user is authenticated
  if (!isAuthenticated()) {
    return <Navigate to={`/login?returnTo=${encodeURIComponent(location.pathname)}`} replace />
  }

  // Check platform owner requirement
  if (requirePlatformOwner && !isPlatformOwner()) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold mb-4">Access Denied</h2>
          <p className="text-muted-foreground">
            You need platform owner privileges to access this page.
          </p>
        </div>
      </div>
    )
  }

  // Check club admin requirement
  if (requireClubAdmin && !isClubAdmin(requireClubAdmin)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold mb-4">Access Denied</h2>
          <p className="text-muted-foreground">
            You need to be an admin of this club to access this page.
          </p>
        </div>
      </div>
    )
  }

  // Check club membership requirement
  if (requireClubMembership && !isClubMember(requireClubMembership)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold mb-4">Access Denied</h2>
          <p className="text-muted-foreground">
            You need to be a member of this club to access this page.
          </p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

export default ProtectedRoute
