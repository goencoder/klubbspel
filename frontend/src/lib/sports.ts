import type {
  Sport,
  SeriesFormat,
  SeriesMatchConfiguration,
  SeriesParticipantMode,
  SeriesScoringProfile
} from '@/types/api'

export const DEFAULT_SPORT: Sport = 'SPORT_TABLE_TENNIS'
export const SUPPORTED_SPORTS: Sport[] = [DEFAULT_SPORT]

export const DEFAULT_SERIES_FORMAT: SeriesFormat = 'SERIES_FORMAT_OPEN_PLAY'
export const SUPPORTED_SERIES_FORMATS: SeriesFormat[] = [DEFAULT_SERIES_FORMAT]

export function sportTranslationKey(sport: Sport): string {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return 'sports.table_tennis'
    case 'SPORT_TENNIS':
      return 'sports.tennis'
    case 'SPORT_PADEL':
      return 'sports.padel'
    case 'SPORT_SOCCER':
      return 'sports.soccer'
    case 'SPORT_FLOORBALL':
      return 'sports.floorball'
    case 'SPORT_BASKETBALL':
      return 'sports.basketball'
    case 'SPORT_ICE_HOCKEY':
      return 'sports.ice_hockey'
    case 'SPORT_FRISBEE_GOLF':
      return 'sports.frisbee_golf'
    case 'SPORT_GOLF':
      return 'sports.golf'
    case 'SPORT_FISHING':
      return 'sports.fishing'
    default:
      return 'sports.unknown'
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

const TABLE_TENNIS_MATCH_CONFIGURATION: SeriesMatchConfiguration = {
  participantMode: 'SERIES_PARTICIPANT_MODE_INDIVIDUAL',
  participantsPerSide: 1,
  scoringProfile: 'SERIES_SCORING_PROFILE_TABLE_TENNIS'
}

export function defaultMatchConfigurationForSport(_sport: Sport): SeriesMatchConfiguration {
  // Currently only table tennis is supported; future sports will map to their
  // respective participant/result profiles.
  return { ...TABLE_TENNIS_MATCH_CONFIGURATION }
}

export function participantModeTranslationKey(mode: SeriesParticipantMode): string {
  switch (mode) {
    case 'SERIES_PARTICIPANT_MODE_TEAM':
      return 'series.matchConfiguration.participantModes.team'
    case 'SERIES_PARTICIPANT_MODE_INDIVIDUAL':
    default:
      return 'series.matchConfiguration.participantModes.individual'
  }
}

export function scoringProfileTranslationKey(profile: SeriesScoringProfile): string {
  switch (profile) {
    case 'SERIES_SCORING_PROFILE_SCORELINE':
      return 'series.matchConfiguration.scoringProfiles.scoreline'
    case 'SERIES_SCORING_PROFILE_STROKE_CARD':
      return 'series.matchConfiguration.scoringProfiles.stroke_card'
    case 'SERIES_SCORING_PROFILE_WEIGH_IN':
      return 'series.matchConfiguration.scoringProfiles.weigh_in'
    case 'SERIES_SCORING_PROFILE_TABLE_TENNIS':
    default:
      return 'series.matchConfiguration.scoringProfiles.table_tennis'
  }
}
