import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useAuthStore } from '@/store/auth'
import { apiClient } from '@/services/api'
import type { Club } from '@/types/api'
import { Check, ChevronDown, Users, UserCheck } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

interface ClubNavigationProps {
  showAllOption?: boolean
  onClubChange?: (clubId: string | null) => void
}

export function ClubNavigation({ showAllOption = true, onClubChange }: ClubNavigationProps) {
  const { t } = useTranslation()
  const { user, selectedClubId, selectClub, isClubMember, isClubAdmin } = useAuthStore()
  const [allClubs, setAllClubs] = useState<Club[]>([])
  const [loading, setLoading] = useState(false)

  // Load all clubs for the dropdown
  useEffect(() => {
    const loadAllClubs = async () => {
      try {
        setLoading(true)
        const response = await apiClient.listClubs({ pageSize: 100 })
        // Sort clubs alphabetically by name
        const sortedClubs = response.items.sort((a, b) => a.name.localeCompare(b.name))
        setAllClubs(sortedClubs)
      } catch (_error) {
        // Failed to load clubs for navigation - silently fail
      } finally {
        setLoading(false)
      }
    }

    loadAllClubs()
  }, [])

  const handleClubSelect = (clubId: string | null) => {
    selectClub(clubId)
    onClubChange?.(clubId)
  }

  // Get display text for current selection
  const getDisplayText = () => {
    if (!selectedClubId) {
      return t('clubs.navigation.allClubs')
    }

    if (selectedClubId === 'my-clubs') {
      return t('clubs.navigation.myClubs')
    }

    // Find the specific club name from all clubs
    const club = allClubs.find(c => c.id === selectedClubId)
    if (club) {
      return club.name
    }

    // Fallback to user memberships if not found in all clubs
    if (user?.memberships) {
      const membership = user.memberships.find(m => m.clubId === selectedClubId)
      if (membership && membership.clubName) {
        return membership.clubName
      }
    }

    // If we can't find the club, fall back to "All Clubs"
    return t('clubs.navigation.allClubs')
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="flex items-center gap-2 min-w-[200px] justify-between">
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <span className="truncate">{getDisplayText()}</span>
          </div>
          <ChevronDown className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[250px]">
        {/* Static filter options */}
        {showAllOption && (
          <DropdownMenuItem
            onClick={() => handleClubSelect(null)}
            className="flex items-center justify-between"
          >
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4" />
              <span>{t('clubs.navigation.allClubs')}</span>
            </div>
            {!selectedClubId && <Check className="h-4 w-4" />}
          </DropdownMenuItem>
        )}

        {user && user.memberships && user.memberships.length > 0 && (
          <DropdownMenuItem
            onClick={() => handleClubSelect('my-clubs')}
            className="flex items-center justify-between"
          >
            <div className="flex items-center gap-2">
              <UserCheck className="h-4 w-4" />
              <span>{t('clubs.navigation.myClubs')}</span>
            </div>
            {selectedClubId === 'my-clubs' && <Check className="h-4 w-4" />}
          </DropdownMenuItem>
        )}

        {/* Show all clubs alphabetically */}
        {!loading && allClubs.length > 0 && (
          <>
            <DropdownMenuSeparator />
            {allClubs.map((club) => {
              const isMember = user ? isClubMember(club.id) : false
              const isAdmin = user ? isClubAdmin(club.id) : false
              return (
                <DropdownMenuItem
                  key={club.id}
                  onClick={() => handleClubSelect(club.id)}
                  className="flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-4 rounded-full bg-primary/20 flex items-center justify-center">
                      <div className="w-2 h-2 rounded-full bg-primary" />
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{club.name}</span>
                      {isMember && (
                        <Badge variant={isAdmin ? "default" : "secondary"} className="text-xs">
                          {isAdmin ? "Admin" : "Member"}
                        </Badge>
                      )}
                    </div>
                  </div>
                  {selectedClubId === club.id && <Check className="h-4 w-4" />}
                </DropdownMenuItem>
              )
            })}
          </>
        )}

        {loading && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled>
              <span className="text-muted-foreground">Loading clubs...</span>
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export default ClubNavigation
