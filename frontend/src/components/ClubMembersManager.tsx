import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import apiClient from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { ApiError } from '@/types/api'
import type {
  ClubMemberInfo,
  InvitePlayerRequest,
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
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<MembershipRole>('MEMBERSHIP_ROLE_MEMBER')
  const [showInviteDialog, setShowInviteDialog] = useState(false)
  const [inviting, setInviting] = useState(false)

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

  const handleInvitePlayer = async () => {
    if (!inviteEmail.trim()) {
      toast.error('Please enter an email address')
      return
    }

    try {
      setInviting(true)
      const request: InvitePlayerRequest = {
        clubId,
        email: inviteEmail.trim(),
        role: inviteRole
      }

      const response = await apiClient.invitePlayer(request)

      if (response.success) {
        toast.success(
          response.invitationSent
            ? `Invitation sent to ${inviteEmail}`
            : `${inviteEmail} has been added to the club`
        )
        setInviteEmail('')
        setInviteRole('MEMBERSHIP_ROLE_MEMBER')
        setShowInviteDialog(false)
        await loadMembers()
      }
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || 'Failed to invite player')
    } finally {
      setInviting(false)
    }
  }

  const handleUpdateRole = async (playerId: string, newRole: MembershipRole) => {
    try {
      const request: UpdateMemberRoleRequest = {
        clubId,
        playerId,
        role: newRole
      }

      await apiClient.updateMemberRole(request)
      toast.success('Member role updated successfully')
      await loadMembers()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || 'Failed to update member role')
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
      toast.error(apiError.message || 'Failed to remove member')
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
        return 'Admin'
      case 'MEMBERSHIP_ROLE_MEMBER':
        return 'Member'
      default:
        return 'Unknown'
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
              Club Members
            </CardTitle>
            <CardDescription>
              Manage members of {clubName} ({members.length} total)
            </CardDescription>
          </div>

          {canManageClub && (
            <Dialog open={showInviteDialog} onOpenChange={setShowInviteDialog}>
              <DialogTrigger asChild>
                <Button className="flex items-center gap-2">
                  <UserPlus className="h-4 w-4" />
                  Invite Player
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Invite Player to {clubName}</DialogTitle>
                  <DialogDescription>
                    Send an invitation email to invite a player to join this club.
                  </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                  <div>
                    <Label htmlFor="email">Email Address</Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="player@example.com"
                      value={inviteEmail}
                      onChange={(e) => setInviteEmail(e.target.value)}
                    />
                  </div>

                  <div>
                    <Label htmlFor="role">Role</Label>
                    <Select value={inviteRole} onValueChange={(value) => setInviteRole(value as MembershipRole)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="MEMBERSHIP_ROLE_MEMBER">Member</SelectItem>
                        <SelectItem value="MEMBERSHIP_ROLE_ADMIN">Admin</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <DialogFooter>
                  <Button
                    variant="outline"
                    onClick={() => setShowInviteDialog(false)}
                    disabled={inviting}
                  >
                    Cancel
                  </Button>
                  <Button onClick={handleInvitePlayer} disabled={inviting}>
                    {inviting ? 'Sending...' : 'Send Invitation'}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
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
                <TableHead>Joined</TableHead>
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
                      <span className="text-sm text-muted-foreground">{member.email}</span>
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
                    <Badge variant={member.membership.active ? 'default' : 'outline'}>
                      {member.membership.active ? 'Active' : 'Inactive'}
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
                                title="Promote to Admin"
                              >
                                <Shield className="h-4 w-4" />
                              </Button>
                            ) : (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleUpdateRole(member.playerId, 'MEMBERSHIP_ROLE_MEMBER')}
                                title="Demote to Member"
                              >
                                <User className="h-4 w-4" />
                              </Button>
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleRemoveMember(member.playerId, member.displayName)}
                              title="Remove from Club"
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
    </Card>
  )
}

export default ClubMembersManager
