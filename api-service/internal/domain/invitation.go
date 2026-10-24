package domain

import "time"

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationRevoked  InvitationStatus = "revoked"
)

const InvitationTTL = 7 * 24 * time.Hour

type Invitation struct {
	ID          string
	ProjectID   string
	ProjectName string
	Email       string
	Role        ProjectRole
	Status      InvitationStatus
	InvitedBy   string
	InviterName string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AcceptedAt  *time.Time
}

func (i Invitation) Expired(now time.Time) bool { return now.After(i.ExpiresAt) }

// CheckAcceptable kiểm tra lời mời còn dùng được và đúng người nhận.
func (i Invitation) CheckAcceptable(actorEmail string, now time.Time) error {
	switch {
	case i.Status != InvitationPending:
		return Conflict("invitation_not_pending", "This invitation is no longer valid")
	case i.Expired(now):
		return Conflict("invitation_expired", "This invitation has expired")
	case NormalizeEmail(actorEmail) != NormalizeEmail(i.Email):
		return Forbidden("invitation_wrong_email", "This invitation was sent to a different email address")
	}
	return nil
}

func ValidateInviteRole(r ProjectRole) error {
	if !r.Valid() {
		return Validation("invalid_project_role", "Role must be owner, editor or viewer")
	}
	return nil
}
