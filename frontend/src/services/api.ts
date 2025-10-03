import { useAppStore } from '@/store'

// Custom error class to identify cancelled requests
export class RequestCancelledError extends Error {
  constructor() {
    super('Request was cancelled')
    this.name = 'RequestCancelledError'
  }
}

import type {
  ApiError,
  Club,
  CreateClubRequest,
  CreateClubResponse,
  CreatePlayerRequest,
  CreatePlayerResponse,
  CreateSeriesRequest,
  DeleteMatchRequest,
  DeleteMatchResponse,
  FindMergeCandidatesRequest,
  FindMergeCandidatesResponse,
  GetLeaderboardRequest,
  GetLeaderboardResponse,
  GetSeriesRulesRequest,
  GetSeriesRulesResponse,
  ListClubsRequest,
  ListClubsResponse,
  ListMatchesRequest,
  ListMatchesResponse,
  ListPlayersRequest,
  ListPlayersResponse,
  ListSeriesRequest,
  ListSeriesResponse,
  MergePlayerRequest,
  MergePlayerResponse,
  Player,
  ReportMatchRequest,
  ReportMatchResponse,
  ReportMatchV2Request,
  ReportMatchV2Response,
  Series,
  UpdateClubRequest,
  UpdateMatchRequest,
  UpdateMatchResponse,
  UpdatePlayerRequest,
  UpdateSeriesRequest
} from '@/types/api'
import type {
  AuthUser,
  AddPlayerToClubRequest,
  AddPlayerToClubResponse,
  InvitePlayerRequest,
  InvitePlayerResponse,
  JoinClubRequest,
  JoinClubResponse,
  LeaveClubRequest,
  LeaveClubResponse,
  ListClubMembersRequest,
  ListClubMembersResponse,
  ListPlayerMembershipsRequest,
  ListPlayerMembershipsResponse,
  SendMagicLinkRequest,
  SendMagicLinkResponse,
  UpdateMemberRoleRequest,
  UpdateMemberRoleResponse,
  ValidateTokenRequest,
  ValidateTokenResponse
} from '@/types/membership'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

class ApiClient {
  private abortControllers = new Map<string, AbortController>()

  private getHeaders(): HeadersInit {
    const { language } = useAppStore.getState()
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      'Accept-Language': language || 'sv',
    }

    // Add Origin header if in development
    if (window.location.origin) {
      headers['Origin'] = window.location.origin
    }

    // Add authentication header if available
    const authState = localStorage.getItem('auth-storage')
    if (authState) {
      try {
        const parsedState = JSON.parse(authState)
        // Zustand persist stores data in a 'state' property
        const user = parsedState.state?.user
        if (user?.apiToken) {
          headers['Authorization'] = `Bearer ${user.apiToken}`
        }
      } catch (_error) {
        // Failed to parse auth state - continue without auth headers
      }
    }

    return headers
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit & { requestId?: string } = {}
  ): Promise<T> {
    const { requestId, ...requestOptions } = options

    // Cancel previous request with same ID if exists
    if (requestId) {
      this.abortControllers.get(requestId)?.abort()
      const controller = new AbortController()
      this.abortControllers.set(requestId, controller)
      requestOptions.signal = controller.signal
    }

    const url = `${BASE_URL}${endpoint}`
    const config: RequestInit = {
      headers: this.getHeaders(),
      ...requestOptions
    }

    try {
      const response = await fetch(url, config)

      // Clean up abort controller
      if (requestId) {
        this.abortControllers.delete(requestId)
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        const apiError: ApiError = {
          code: errorData.code || `HTTP_${response.status}`,
          message: errorData.message || response.statusText,
          details: errorData.details
        }
        throw Object.assign(new Error(apiError.message), apiError)
      }

      // Handle empty responses (like DELETE)
      if (response.status === 204 || response.headers.get('content-length') === '0') {
        return {} as T
      }

      const responseData = await response.json()
      return responseData
    } catch (error) {
      // Clean up abort controller on error
      if (requestId) {
        this.abortControllers.delete(requestId)
      }

      // Silently ignore abort errors - these are expected during navigation/component unmounting
      if (error instanceof Error && error.name === 'AbortError') {
        return Promise.reject(new RequestCancelledError())
      }

      // Re-throw API errors
      if (error && typeof error === 'object' && 'code' in error) {
        throw error
      }

      // Network or other errors
      const apiError: ApiError = {
        code: 'NETWORK_ERROR',
        message: error instanceof Error ? error.message : 'Network error occurred'
      }
      throw Object.assign(new Error(apiError.message), apiError)
    }
  }

  private get<T>(endpoint: string, requestId?: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET', requestId })
  }

  private post<T>(endpoint: string, data?: unknown, requestId?: string): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      requestId
    })
  }

  private patch<T>(endpoint: string, data: unknown, requestId?: string): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: JSON.stringify(data),
      requestId
    })
  }

  private delete<T>(endpoint: string, requestId?: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE', requestId })
  }

  // Club API methods
  async listClubs(params: ListClubsRequest = {}, requestId?: string): Promise<ListClubsResponse> {
    const searchParams = new URLSearchParams()
    // Use parameter names as specified in the OpenAPI spec
    if (params.searchQuery) {searchParams.append('searchQuery', params.searchQuery)}
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.cursorAfter) {searchParams.append('cursorAfter', params.cursorAfter)}
    if (params.cursorBefore) {searchParams.append('cursorBefore', params.cursorBefore)}

    const query = searchParams.toString()
    return this.get<ListClubsResponse>(`/v1/clubs${query ? `?${query}` : ''}`, requestId)
  }

  async getClub(id: string): Promise<Club> {
    const response = await this.get<{ club: Club }>(`/v1/clubs/${id}`)
    return response.club
  }

  async createClub(data: CreateClubRequest): Promise<Club> {
    const response = await this.post<CreateClubResponse>('/v1/clubs', data)
    return response.club
  }

  async updateClub(id: string, data: UpdateClubRequest): Promise<Club> {
    const response = await this.patch<{ club: Club }>(`/v1/clubs/${id}`, data)
    return response.club
  }

  async deleteClub(id: string): Promise<void> {
    await this.delete<{ success: boolean }>(`/v1/clubs/${id}`)
  }

  // Player API methods
  async listPlayers(params: ListPlayersRequest = {}, requestId?: string): Promise<ListPlayersResponse> {
    const searchParams = new URLSearchParams()
    // Use parameter names as specified in the OpenAPI spec
    if (params.searchQuery) {searchParams.append('searchQuery', params.searchQuery)}
    if (params.clubId) {searchParams.append('clubId', params.clubId)}
    if (params.clubFilter && params.clubFilter.length > 0) {
      params.clubFilter.forEach(clubId => {
        searchParams.append('clubFilter', clubId)
      })
    }
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.cursorAfter) {searchParams.append('cursorAfter', params.cursorAfter)}
    if (params.cursorBefore) {searchParams.append('cursorBefore', params.cursorBefore)}

    const query = searchParams.toString()
    return this.get<ListPlayersResponse>(`/v1/players${query ? `?${query}` : ''}`, requestId)
  }

  async getPlayer(id: string): Promise<Player> {
    const response = await this.get<{ player: Player }>(`/v1/players/${id}`)
    return response.player
  }

  async createPlayer(data: CreatePlayerRequest): Promise<CreatePlayerResponse> {
    return this.post<CreatePlayerResponse>('/v1/players', data)
  }

  async updatePlayer(id: string, data: UpdatePlayerRequest): Promise<Player> {
    const response = await this.patch<{ player: Player }>(`/v1/players/${id}`, data)
    return response.player
  }

  async deletePlayer(id: string): Promise<void> {
    await this.delete<{ success: boolean }>(`/v1/players/${id}`)
  }

  async mergePlayer(targetPlayerId: string, data: MergePlayerRequest): Promise<MergePlayerResponse> {
    return this.post<MergePlayerResponse>(`/v1/players/${targetPlayerId}/merge`, data)
  }

  async findMergeCandidates(params: FindMergeCandidatesRequest = {}): Promise<FindMergeCandidatesResponse> {
    const searchParams = new URLSearchParams()
    if (params.clubId) {
      searchParams.append('club_id', params.clubId)
    }
    if (params.namePattern) {
      searchParams.append('name_pattern', params.namePattern)
    }
    
    const query = searchParams.toString()
    return this.get<FindMergeCandidatesResponse>(`/v1/players/merge-candidates${query ? `?${query}` : ''}`)
  }

  // Series API methods
  async listSeries(params: ListSeriesRequest = {}, requestId?: string): Promise<ListSeriesResponse> {
    const searchParams = new URLSearchParams()
    // Use parameter names as specified in the OpenAPI spec
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.cursorAfter) {searchParams.append('cursorAfter', params.cursorAfter)}
    if (params.cursorBefore) {searchParams.append('cursorBefore', params.cursorBefore)}
    if (params.sportFilter && params.sportFilter !== 'SPORT_UNSPECIFIED') {
      searchParams.append('sportFilter', params.sportFilter)
    }
    if (params.clubFilter && params.clubFilter.length > 0) {
      params.clubFilter.forEach(clubId => {
        searchParams.append('clubFilter', clubId)
      })
    }

    const query = searchParams.toString()
    const response = await this.get<{ items: Series[], startCursor?: string, endCursor?: string, hasNextPage: boolean, hasPreviousPage: boolean }>(`/v1/series${query ? `?${query}` : ''}`, requestId)

    return {
      items: response.items,
      startCursor: response.startCursor,
      endCursor: response.endCursor,
      hasNextPage: response.hasNextPage,
      hasPreviousPage: response.hasPreviousPage
    }
  }

  async getSeries(id: string): Promise<Series> {
    const response = await this.get<{ series: Series }>(`/v1/series/${id}`)
    return response.series
  }

  async createSeries(data: CreateSeriesRequest): Promise<Series> {
    const response = await this.post<{ series: Series }>('/v1/series', data)
    return response.series
  }

  async updateSeries(id: string, data: UpdateSeriesRequest): Promise<Series> {
    const response = await this.patch<{ series: Series }>(`/v1/series/${id}`, data)
    return response.series
  }

  async deleteSeries(id: string): Promise<void> {
    await this.delete<{ success: boolean }>(`/v1/series/${id}`)
  }

  async getSeriesRules(params: GetSeriesRulesRequest): Promise<GetSeriesRulesResponse> {
    const searchParams = new URLSearchParams()
    searchParams.append('format', params.format)
    if (params.ladderRules) {
      searchParams.append('ladder_rules', params.ladderRules)
    }
    
    const query = searchParams.toString()
    return this.get<GetSeriesRulesResponse>(`/v1/series/rules${query ? `?${query}` : ''}`)
  }

  // Match API methods
  async listMatches(params: ListMatchesRequest, requestId?: string): Promise<ListMatchesResponse> {
    const searchParams = new URLSearchParams()
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.cursorAfter) {searchParams.append('cursorAfter', params.cursorAfter)}
    if (params.cursorBefore) {searchParams.append('cursorBefore', params.cursorBefore)}

    const query = searchParams.toString()
    return this.get<ListMatchesResponse>(
      `/v1/series/${params.seriesId}/matches${query ? `?${query}` : ''}`,
      requestId
    )
  }

  async reportMatch(data: ReportMatchRequest): Promise<ReportMatchResponse> {
    return this.post<ReportMatchResponse>('/v1/matches:report', data)
  }

  async reportMatchV2(data: ReportMatchV2Request): Promise<ReportMatchV2Response> {
    return this.post<ReportMatchV2Response>('/v2/matches:report', data)
  }

  async updateMatch(data: UpdateMatchRequest): Promise<UpdateMatchResponse> {
    return this.patch<UpdateMatchResponse>(`/v1/matches/${data.matchId}`, data)
  }

  async deleteMatch(data: DeleteMatchRequest): Promise<DeleteMatchResponse> {
    return this.delete<DeleteMatchResponse>(`/v1/matches/${data.matchId}`)
  }

  // Leaderboard API methods
  async getLeaderboard(params: GetLeaderboardRequest, requestId?: string): Promise<GetLeaderboardResponse> {
    const searchParams = new URLSearchParams()
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.cursorAfter) {searchParams.append('cursorAfter', params.cursorAfter)}
    if (params.cursorBefore) {searchParams.append('cursorBefore', params.cursorBefore)}

    const query = searchParams.toString()
    return this.get<GetLeaderboardResponse>(
      `/v1/series/${params.seriesId}/leaderboard${query ? `?${query}` : ''}`,
      requestId
    )
  }

  // === CLUB MEMBERSHIP API METHODS ===

  // Join a club (self-registration)
  async joinClub(data: JoinClubRequest): Promise<JoinClubResponse> {
    return this.post<JoinClubResponse>(`/v1/clubs/${data.clubId}/members`, {})
  }

  // Leave a club
  async leaveClub(data: LeaveClubRequest): Promise<LeaveClubResponse> {
    return this.delete<LeaveClubResponse>(`/v1/clubs/${data.clubId}/members/${data.playerId}`)
  }

  // Invite a player to join a club (admin only)
  async invitePlayer(data: InvitePlayerRequest): Promise<InvitePlayerResponse> {
    return this.post<InvitePlayerResponse>(`/v1/clubs/${data.clubId}/invitations`, {
      email: data.email,
      role: data.role || 'MEMBERSHIP_ROLE_MEMBER'
    })
  }

  // Add a player to a club (admin only)
  async addPlayerToClub(data: AddPlayerToClubRequest): Promise<AddPlayerToClubResponse> {
    return this.post<AddPlayerToClubResponse>(`/v1/clubs/${data.clubId}/players`, {
      firstName: data.firstName,
      lastName: data.lastName,
      email: data.email || undefined
    })
  }

  // Update a member's role (promote/demote)
  async updateMemberRole(data: UpdateMemberRoleRequest): Promise<UpdateMemberRoleResponse> {
    return this.patch<UpdateMemberRoleResponse>(`/v1/clubs/${data.clubId}/members/${data.playerId}/role`, {
      role: data.role
    })
  }

  // List members of a club
  async listClubMembers(params: ListClubMembersRequest, requestId?: string): Promise<ListClubMembersResponse> {
    const searchParams = new URLSearchParams()
    if (params.pageSize) {searchParams.append('pageSize', params.pageSize.toString())}
    if (params.pageToken) {searchParams.append('pageToken', params.pageToken)}
    if (params.activeOnly !== undefined) {searchParams.append('activeOnly', params.activeOnly.toString())}

    const query = searchParams.toString()
    return this.get<ListClubMembersResponse>(
      `/v1/clubs/${params.clubId}/members${query ? `?${query}` : ''}`,
      requestId
    )
  }

  // List a player's club memberships
  async listPlayerMemberships(params: ListPlayerMembershipsRequest, requestId?: string): Promise<ListPlayerMembershipsResponse> {
    const searchParams = new URLSearchParams()
    if (params.activeOnly !== undefined) {searchParams.append('activeOnly', params.activeOnly.toString())}

    const query = searchParams.toString()
    return this.get<ListPlayerMembershipsResponse>(
      `/v1/players/${params.playerId}/memberships${query ? `?${query}` : ''}`,
      requestId
    )
  }

  // === AUTHENTICATION API METHODS ===

  // Send magic link for authentication
  async sendMagicLink(data: SendMagicLinkRequest): Promise<SendMagicLinkResponse> {
    return this.post<SendMagicLinkResponse>('/v1/auth/magic-link', data)
  }

  // Validate authentication token
  async validateToken(data: ValidateTokenRequest): Promise<ValidateTokenResponse> {
    return this.post<ValidateTokenResponse>('/v1/auth/validate', data)
  }

  // Update current user's profile (first name and last name)
  async updateProfile(data: { firstName: string; lastName: string }): Promise<void> {
    await this.patch<{ success: boolean }>('/v1/auth/profile', {
      first_name: data.firstName,
      last_name: data.lastName
    })
  }

  // Get current user information
  async getCurrentUser(): Promise<{ user: AuthUser }> {
    return this.get<{ user: AuthUser }>('/v1/auth/me')
  }

  // Utility method to cancel specific requests
  cancelRequest(requestId: string): void {
    this.abortControllers.get(requestId)?.abort()
    this.abortControllers.delete(requestId)
  }

  // Cancel all pending requests
  cancelAllRequests(): void {
    this.abortControllers.forEach(controller => controller.abort())
    this.abortControllers.clear()
  }
}

// Utility function to check if an error is a request cancellation
export function isRequestCancelled(error: unknown): boolean {
  return error instanceof RequestCancelledError || 
         (error instanceof Error && error.message === 'Request was cancelled')
}

// Utility function to handle API errors in components, filtering out cancellation errors
export function handleApiError(error: unknown, onError?: (error: ApiError) => void): void {
  if (isRequestCancelled(error)) {
    // Silently ignore cancellation errors
    return
  }
  
  const apiError = error as ApiError
  onError?.(apiError)
}

export const apiClient = new ApiClient()
export default apiClient