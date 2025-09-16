export interface Player {
  id: string
  displayName: string
  normalizedKey: string
  active: boolean
  createdAt: string
}

export interface Series {
  id: string
  clubName: string
  title: string
  startsAt: string
  endsAt: string
  createdAt: string
}

export interface Match {
  id: string
  seriesId: string
  playerAId: string
  playerBId: string
  playerAName: string
  playerBName: string
  scoreA: number
  scoreB: number
  playedAt: string
  createdAt: string
}

export interface PlayerRating {
  seriesId: string
  playerId: string
  playerName: string
  rating: number
  games: number
  wins: number
  losses: number
}

export interface LeaderboardRow {
  rank: number
  playerId: string
  displayName: string
  rating: number
  games: number
  wins: number
  losses: number
  winRate: number
}

export interface CreateSeriesRequest {
  clubName: string
  title: string
  startsAt: string
  endsAt: string
}

export interface CreatePlayerRequest {
  displayName: string
}

export interface CreatePlayerResponse {
  player: Player
  similar: Player[]
}

export interface ReportMatchRequest {
  seriesId: string
  playerAId: string
  playerBId: string
  scoreA: number
  scoreB: number
  playedAt: string
}