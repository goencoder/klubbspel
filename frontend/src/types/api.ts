// Generated types based on the Klubbspel API schema
// This would normally be auto-generated from the OpenAPI spec

export interface ApiError {
  code: string
  message: string
  details?: unknown[]
}

export interface ApiResponse<T> {
  data?: T
  error?: ApiError
}

// Common types
export interface Id {
  id: string
}

export interface TimeRange {
  start: string // ISO timestamp
  end: string   // ISO timestamp
}

export type Sport =
  | 'SPORT_UNSPECIFIED'
  | 'SPORT_TABLE_TENNIS'
  | 'SPORT_TENNIS'
  | 'SPORT_PADEL'

export type SeriesFormat =
  | 'SERIES_FORMAT_UNSPECIFIED'
  | 'SERIES_FORMAT_OPEN_PLAY'
  | 'SERIES_FORMAT_LADDER'
  | 'SERIES_FORMAT_CUP'

// Club types
export interface Club {
  id: string
  name: string
  supportedSports: Sport[]
}

export interface CreateClubRequest {
  name: string
  supportedSports?: Sport[]
}

export interface CreateClubResponse {
  club: Club
}

export interface UpdateClubRequest {
  name?: string
  supportedSports?: Sport[]
}

export interface ListClubsRequest {
  searchQuery?: string
  pageSize?: number
  cursorAfter?: string
  cursorBefore?: string
}

export interface ListClubsResponse {
  items: Club[]
  startCursor?: string
  endCursor?: string
  hasNextPage: boolean
  hasPreviousPage: boolean
}

// Player types
export interface Player {
  id: string
  displayName: string  // Backend uses camelCase per swagger
  normalizedKey: string  // Backend uses camelCase per swagger
  clubId: string  // DEPRECATED: Use clubMemberships instead
  active: boolean
  // NEW: Multi-club support
  email: string
  firstName: string
  lastName: string
  clubMemberships: ClubMembership[]
  isPlatformOwner: boolean
  lastLoginAt?: string
}

// Club membership for players
export interface ClubMembership {
  clubId: string
  role: 'MEMBERSHIP_ROLE_MEMBER' | 'MEMBERSHIP_ROLE_ADMIN' | 'MEMBERSHIP_ROLE_UNSPECIFIED' // i18n-ignore: API constants
  active: boolean
  joinedAt: string
  leftAt?: string
}

export interface CreatePlayerRequest {
  displayName: string  // Frontend uses camelCase for requests
  initialClubId: string
}

export interface CreatePlayerResponse {
  player: Player
  similar: Player[]
}

export interface UpdatePlayerRequest {
  displayName?: string
  firstName?: string
  lastName?: string
  clubId?: string
  active?: boolean
}

export interface ListPlayersRequest {
  searchQuery?: string
  clubId?: string // DEPRECATED: Use clubFilter instead
  clubFilter?: string[] // Support multiple club IDs and special "OPEN" value
  pageSize?: number
  cursorAfter?: string
  cursorBefore?: string
}

export interface ListPlayersResponse {
  items: Player[]
  startCursor?: string
  endCursor?: string
  hasNextPage: boolean
  hasPreviousPage: boolean
}

export interface MergePlayerRequest {
  sourcePlayerId: string  // The player to merge from (usually non-authenticated)
}

export interface MergePlayerResponse {
  player: Player         // The resulting merged player
  matchesUpdated: number // Number of matches that were updated
  tokensUpdated: number  // Number of tokens that were updated
}

export interface FindMergeCandidatesRequest {
  clubId?: string      // Optional club ID to limit search to specific club
  namePattern?: string // Optional name pattern to search for
}

export interface MergeCandidate {
  player: Player        // The email-less player that could be merged
  similarityScore: number // Similarity score between candidate and authenticated user (0.0 to 1.0)
}

export interface FindMergeCandidatesResponse {
  candidates: MergeCandidate[] // List of email-less players with similarity scores
}

// Series types
export type SeriesVisibility = 'SERIES_VISIBILITY_OPEN' | 'SERIES_VISIBILITY_CLUB_ONLY' | 'SERIES_VISIBILITY_UNSPECIFIED'

export interface Series {
  id: string
  clubId?: string
  title: string
  startsAt: string  // Backend sends startsAt per swagger
  endsAt: string    // Backend sends endsAt per swagger
  visibility: SeriesVisibility
  sport: Sport
  format: SeriesFormat
}

export interface CreateSeriesRequest {
  clubId?: string
  title: string
  startsAt: string
  endsAt: string
  visibility: SeriesVisibility
  sport?: Sport
  format?: SeriesFormat
}

export interface UpdateSeriesRequest {
  title?: string
  startsAt?: string
  endsAt?: string
  visibility?: SeriesVisibility
  sport?: Sport
  format?: SeriesFormat
}

export interface ListSeriesRequest {
  pageSize?: number
  cursorAfter?: string
  cursorBefore?: string
  sportFilter?: Sport
  clubFilter?: string[]  // Club IDs to filter by, special value "OPEN" for open series
}

export interface ListSeriesResponse {
  items: Series[]
  startCursor?: string
  endCursor?: string
  hasNextPage: boolean
  hasPreviousPage: boolean
}

// Match types
export interface MatchView {
  id: string
  seriesId: string
  playerAName: string
  playerBName: string
  scoreA: number
  scoreB: number
  playedAt: string
}

export interface ReportMatchRequest {
  seriesId: string
  playerAId: string
  playerBId: string
  scoreA: number
  scoreB: number
  playedAt: string
}

export interface ReportMatchResponse {
  matchId: string
}

export interface ListMatchesRequest {
  seriesId: string
  pageSize?: number
  cursorAfter?: string
  cursorBefore?: string
}

export interface ListMatchesResponse {
  items: MatchView[]
  startCursor?: string
  endCursor?: string
  hasNextPage: boolean
  hasPreviousPage: boolean
}

export interface UpdateMatchRequest {
  matchId: string
  scoreA?: number
  scoreB?: number
  playedAt?: string
}

export interface UpdateMatchResponse {
  match: MatchView
}

export interface DeleteMatchRequest {
  matchId: string
}

export interface DeleteMatchResponse {
  // Empty response
}

export interface ReorderMatchesRequest {
  seriesId: string
  matchIds: string[]
}

export interface ReorderMatchesResponse {
  matches: MatchView[]
}

// Leaderboard types
export interface LeaderboardEntry {
  rank: number
  playerId: string
  playerName: string
  eloRating: number
  matchesPlayed: number
  matchesWon: number
  matchesLost: number
  winRate: number
  gamesWon: number
  gamesLost: number
  gameWinRate: number
  rankChange: number
}

export interface GetLeaderboardRequest {
  seriesId: string
  pageSize?: number
  cursorAfter?: string
  cursorBefore?: string
}

export interface GetLeaderboardResponse {
  entries: LeaderboardEntry[]
  startCursor?: string
  endCursor?: string
  hasNextPage: boolean
  hasPreviousPage: boolean
  totalPlayers: number
  lastUpdated: string
}