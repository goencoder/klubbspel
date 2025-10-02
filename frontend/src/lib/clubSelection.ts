import type { Club } from '@/types/api'

interface DeriveClubIdOptions {
  manageableClubs: Club[]
  selectedClubId?: string | null
  previousClubId: string
}

export function deriveAutomaticClubId({
  manageableClubs,
  selectedClubId,
  previousClubId,
}: DeriveClubIdOptions): string {
  if (
    selectedClubId &&
    manageableClubs.some((club) => club.id === selectedClubId)
  ) {
    return selectedClubId
  }

  if (manageableClubs.length === 1) {
    return manageableClubs[0].id
  }

  if (
    previousClubId &&
    manageableClubs.some((club) => club.id === previousClubId)
  ) {
    return previousClubId
  }

  return ''
}
