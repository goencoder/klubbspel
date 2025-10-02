import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { AddPlayerDialog } from '@/components/AddPlayerDialog'
import apiClient from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { ApiError } from '@/types/api'
import type {
  ClubMemberInfo,
  MembershipRole,
  UpdateMemberRoleRequest
} from '@/types/membership'
import { Mail, Shield, User, UserPlus, UserX, Users } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

interface ClubMembersManagerProps {
  clubId: string
  clubName: string
}

export function ClubMembersManager({ clubId, clubName }: ClubMembersManagerProps) {
  const { t } = useTranslation()
  const { user, isClubAdmin, isPlatformOwner } = useAuthStore()
  const [members, setMembers] = useState<ClubMemberInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddPlayerDialog, setShowAddPlayerDialog] = useState(false)

  const canManageClub = isClubAdmin(clubId) || isPlatformOwner()

  const loadMembers = useCallback(async () => {
    try {
      setLoading(true)
      const response = await apiClient.listClubMembers({
        clubId,
        pageSize: 100,
        activeOnly: false
      })
      setMembers(response.members)
    } catch (error) {
      const apiError = error as ApiError
      toast.error(t('error.loadClubMembers', {
        message: apiError.message || 'Failed to load members'
      }))
    } finally {
      setLoading(false)
    }
  }, [clubId, t])

  useEffect(() => {
    loadMembers()
  }, [loadMembers])

  const handlePlayerAdded = () => {
    // Reload members list after adding a player
    loadMembers()
  }

  const handleUpdateRole = async (playerId: string, newRole: MembershipRole) => {
    try {
      const request: UpdateMemberRoleRequest = {
        clubId,
        playerId,
        role: newRole
      }

      await apiClient.updateMemberRole(request)
      toast.success(t('clubs.members.roleUpdated'))
      await loadMembers()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('clubs.members.roleUpdateFailed'))
    }
  }

  const handleRemoveMember = async (playerId: string, playerName: string) => {
    if (!window.confirm(`Are you sure you want to remove ${playerName} from ${clubName}?`)) {
      return
    }

    try {
      await apiClient.leaveClub({ clubId, playerId })
      toast.success(`${playerName} has been removed from the club`)
      await loadMembers()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('clubs.members.removeFailed'))
    }
  }

  const getRoleBadgeVariant = (role: MembershipRole) => {
    switch (role) {
      case 'MEMBERSHIP_ROLE_ADMIN':
        return 'default'
      case 'MEMBERSHIP_ROLE_MEMBER':
        return 'secondary'
      default:
        return 'outline'
    }
  }

  const getRoleDisplayName = (role: MembershipRole) => {
    switch (role) {
      case 'MEMBERSHIP_ROLE_ADMIN':
        return t('clubs.members.admin')
      case 'MEMBERSHIP_ROLE_MEMBER':
        return t('clubs.members.member')
      default:
        return t('clubs.members.unknown')
    }
  }

  if (!user) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              {t('clubs.members.title')}
            </CardTitle>
            <CardDescription>
              {t('clubs.members.manage')} {clubName} ({members.length} {t('clubs.members.total')})
            </CardDescription>
          </div>

          {canManageClub && (
            <Button 
              className="flex items-center gap-2"
              onClick={() => setShowAddPlayerDialog(true)}
            >
              <UserPlus className="h-4 w-4" />
              {t('clubs.members.addPlayer')}
            </Button>
          )}
        </div>
      </CardHeader>

      <CardContent>
        {loading ? (
          <div className="text-center py-8">Loading members...</div>
        ) : members.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No members found
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Member</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>{t('clubs.members.joined')}</TableHead>
                <TableHead>Status</TableHead>
                {canManageClub && <TableHead>Actions</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {members.map((member) => (
                <TableRow key={member.playerId}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <User className="h-4 w-4" />
                      <span className="font-medium">{member.displayName}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Mail className="h-4 w-4 text-muted-foreground" />
                      {member.email ? (
                        <span className="text-sm text-muted-foreground">{member.email}</span>
                      ) : (
                        <span className="text-sm text-muted-foreground italic">No email (club-created)</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getRoleBadgeVariant(member.membership.role)}>
                      {getRoleDisplayName(member.membership.role)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {new Date(member.membership.joinedAt).toLocaleDateString()}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={member.membership.role === 'MEMBERSHIP_ROLE_ADMIN' ? 'default' : 'secondary'}>
                      {member.membership.role === 'MEMBERSHIP_ROLE_ADMIN' ? t('clubs.members.admin') : t('clubs.members.member')}
                    </Badge>
                  </TableCell>
                  {canManageClub && (
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {member.playerId !== user.playerId && (
                          <>
                            {member.membership.role === 'MEMBERSHIP_ROLE_MEMBER' ? (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleUpdateRole(member.playerId, 'MEMBERSHIP_ROLE_ADMIN')}
                                title={t('clubs.members.promoteToAdmin')}
                              >
                                <Shield className="h-4 w-4" />
                              </Button>
                            ) : (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleUpdateRole(member.playerId, 'MEMBERSHIP_ROLE_MEMBER')}
                                title={t('clubs.members.demoteToMember')}
                              >
                                <User className="h-4 w-4" />
                              </Button>
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleRemoveMember(member.playerId, member.displayName)}
                              title={t('clubs.members.removeFromClub')}
                            >
                              <UserX className="h-4 w-4" />
                            </Button>
                          </>
                        )}
                      </div>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
      
      <AddPlayerDialog
        open={showAddPlayerDialog}
        onOpenChange={setShowAddPlayerDialog}
        clubId={clubId}
        clubName={clubName}
        onPlayerAdded={handlePlayerAdded}
      />
    </Card>
  )
}

export default ClubMembersManager
