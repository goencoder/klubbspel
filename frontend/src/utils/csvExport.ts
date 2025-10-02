/**
 * CSV Export utilities for Klubbspel
 */

export interface MatchCSVData extends Record<string, unknown> {
  sequence: number
  playerA: string
  scoreA: number
  scoreB: number
  playerB: string
  winner: string
  date: string
  time: string
  playedAt: string
}

export interface LeaderboardCSVData extends Record<string, unknown> {
  rank: number
  player: string
  rating: number
  games: number
  wins: number
  losses: number
  winRate: string
}

/**
 * Convert array of objects to CSV string
 */
function arrayToCSV<T extends Record<string, unknown>>(data: T[], headers: string[]): string {
  const csvHeaders = headers.join(',')
  const csvRows = data.map(row => 
    headers.map(header => {
      const value = row[header]
      // Escape quotes and wrap in quotes if contains comma, quote, or newline
      const stringValue = String(value ?? '')
      if (stringValue.includes(',') || stringValue.includes('"') || stringValue.includes('\n')) {
        return `"${stringValue.replace(/"/g, '""')}"`
      }
      return stringValue
    }).join(',')
  )
  
  return [csvHeaders, ...csvRows].join('\n')
}

/**
 * Download CSV file
 */
function downloadCSV(csvContent: string, filename: string): void {
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  
  link.setAttribute('href', url)
  link.setAttribute('download', filename)
  link.style.visibility = 'hidden'
  
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  
  URL.revokeObjectURL(url)
}

/**
 * Export matches to CSV
 */
export function exportMatchesToCSV(matches: MatchCSVData[], seriesName?: string): void {
  const headers = ['sequence', 'playerA', 'scoreA', 'scoreB', 'playerB', 'winner', 'date', 'time', 'playedAt']
  const csvContent = arrayToCSV(matches, headers)
  
  const filename = seriesName 
    ? `matches_${seriesName.replace(/[^a-zA-Z0-9]/g, '_')}_${new Date().toISOString().split('T')[0]}.csv`
    : `matches_${new Date().toISOString().split('T')[0]}.csv`
  
  downloadCSV(csvContent, filename)
}

/**
 * Export leaderboard to CSV
 */
export function exportLeaderboardToCSV(leaderboard: LeaderboardCSVData[], seriesName?: string): void {
  const headers = ['rank', 'player', 'rating', 'games', 'wins', 'losses', 'winRate']
  const csvContent = arrayToCSV(leaderboard, headers)
  
  const filename = seriesName 
    ? `leaderboard_${seriesName.replace(/[^a-zA-Z0-9]/g, '_')}_${new Date().toISOString().split('T')[0]}.csv`
    : `leaderboard_${new Date().toISOString().split('T')[0]}.csv`
  
  downloadCSV(csvContent, filename)
}