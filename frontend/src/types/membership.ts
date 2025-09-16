// Club membership types for Phase 4 implementation
// This file extends the existing types to support multi-club functionality


// Membership roles
export type MembershipRole = 'MEMBERSHIP_ROLE_MEMBER' | 'MEMBERSHIP_ROLE_ADMIN' | 'MEMBERSHIP_ROLE_UNSPECIFIED'

// Club membership information
export interface ClubMembership {
  clubId: string
  role: MembershipRole
  joinedAt: string // ISO timestamp
}

// Club member information (for viewing members of a club)
export interface ClubMemberInfo {
  playerId: string
  displayName: string
  email: string
  membership: ClubMembership
}

// Player membership information (for viewing clubs a player belongs to)
export interface PlayerMembershipInfo {
  clubId: string
  clubName: string
  membership: ClubMembership
}

// === JOIN CLUB ===
export interface JoinClubRequest {
  clubId: string
}

export interface JoinClubResponse {
  success: boolean
  membership: ClubMembership
}

// === LEAVE CLUB ===
export interface LeaveClubRequest {
  clubId: string
  playerId: string
}

export interface LeaveClubResponse {
  success: boolean
}

// === INVITE PLAYER ===
export interface InvitePlayerRequest {
  clubId: string
  email: string
  role?: MembershipRole
}

export interface InvitePlayerResponse {
  success: boolean
  invitationSent: boolean
}

// === UPDATE MEMBER ROLE ===
export interface UpdateMemberRoleRequest {
  clubId: string
  playerId: string
  role: MembershipRole
}

export interface UpdateMemberRoleResponse {
  success: boolean
  membership: ClubMembership
}

// === LIST CLUB MEMBERS ===
export interface ListClubMembersRequest {
  clubId: string
  pageSize?: number
  pageToken?: string
  activeOnly?: boolean
}

export interface ListClubMembersResponse {
  members: ClubMemberInfo[]
  nextPageToken?: string
}

// === LIST PLAYER MEMBERSHIPS ===
export interface ListPlayerMembershipsRequest {
  playerId: string
  activeOnly?: boolean
}

export interface ListPlayerMembershipsResponse {
  memberships: PlayerMembershipInfo[]
}

// === AUTHENTICATION TYPES ===

// Magic link authentication
export interface SendMagicLinkRequest {
  email: string
  returnUrl?: string
}

export interface SendMagicLinkResponse {
  success: boolean
  message: string
}

export interface ValidateTokenRequest {
  token: string
}

export interface ValidateTokenResponse {
  apiToken: string
  user: {
    id: string
    email: string
    firstName?: string
    lastName?: string
    clubMemberships?: ClubMembership[]
    isPlatformOwner?: boolean
    lastLoginAt?: string | null
  }
  expiresAt: string
}

// Current user context
export interface CurrentUser {
  email: string
  displayName: string
  playerId: string
  apiToken: string
  memberships: ClubMembership[]
  isPlatformOwner: boolean
}

// Auth user from backend (matches protobuf AuthUser)
export interface AuthUser {
  id: string
  email: string
  firstName: string
  lastName: string
  clubMemberships: ClubMembership[]
  isPlatformOwner: boolean
  lastLoginAt?: string
}

// === AUTHORIZATION HELPERS ===
export interface AuthContext {
  user: CurrentUser | null
  selectedClubId: string | null
  isAuthenticated: boolean
  isPlatformOwner: boolean
  isClubAdmin: (clubId: string) => boolean
  isClubMember: (clubId: string) => boolean
}
