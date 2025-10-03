import type { Sport, SeriesFormat } from '@/types/api'
import type { LucideIcon } from 'lucide-react'
import { Circle, CircleDot } from 'lucide-react'

export const DEFAULT_SPORT: Sport = 'SPORT_TABLE_TENNIS'
export const SUPPORTED_SPORTS: Sport[] = [DEFAULT_SPORT, 'SPORT_TENNIS']

export const DEFAULT_SERIES_FORMAT: SeriesFormat = 'SERIES_FORMAT_OPEN_PLAY'
export const SUPPORTED_SERIES_FORMATS: SeriesFormat[] = [DEFAULT_SERIES_FORMAT]

/**
 * Returns the i18n translation key for a given sport.
 * Keys should match those defined in the i18n locale files.
 */
export function sportTranslationKey(sport: Sport): string {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return 'sports.table_tennis'
    case 'SPORT_TENNIS':
      return 'sports.tennis'
    default:
      return 'sports.unknown'
  }
}

export function sportIconComponent(sport: Sport): LucideIcon {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return CircleDot  // Represents ping pong ball
    case 'SPORT_TENNIS':
      return Circle     // Represents tennis ball
    default:
      return Circle
  }
}

export function seriesFormatTranslationKey(format: SeriesFormat): string {
  switch (format) {
    case 'SERIES_FORMAT_OPEN_PLAY':
      return 'series.formatOptions.open_play'
    case 'SERIES_FORMAT_LADDER':
      return 'series.formatOptions.ladder'
    case 'SERIES_FORMAT_CUP':
      return 'series.formatOptions.cup'
    default:
      return 'series.formatOptions.unknown'
  }
}
