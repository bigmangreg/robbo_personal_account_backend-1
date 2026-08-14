package achievements

import (
	"time"
)

const (
	EventLogin            = "login"
	EventProfileUpdate    = "profile_update"
	EventProjectCreate    = "project_create"
	EventProjectPublish   = "project_publish"
	EventReactionPut      = "reaction_put"
	EventReactionReceived = "reaction_received"
)

type EvaluatePayload struct {
	FullName           string
	TargetUserID       string // for reaction_received: project owner
	ReactionCountOnOwn int64  // optional override
}

type AchievementView struct {
	Code           string
	TitleKey       string
	DescriptionKey string
	Icon           string
	Category       string
	GrantedAt      *time.Time
	IsEarned       bool
}

type UserAchievementsView struct {
	UserID    string
	Earned    []AchievementView
	Available []AchievementView
}

type UseCase interface {
	Evaluate(userID, event string, payload EvaluatePayload) error
	ListForUser(userID string) (*UserAchievementsView, error)
	ListUnseen(userID string) ([]UserAchievement, error)
	MarkSeen(userID string, codes []string) error
	ListDefinitions() ([]Definition, error)
	UpsertDefinition(input UpsertDefinitionInput) (*Definition, error)
	SetDefinitionEnabled(code string, enabled bool) (*Definition, error)
}
