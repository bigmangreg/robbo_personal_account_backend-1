package achievements

import (
	"encoding/json"
	"time"
)

type Definition struct {
	Code           string
	TitleKey       string
	DescriptionKey string
	Icon           string
	Category       string
	CriteriaJSON   json.RawMessage
	IsEnabled      bool
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserAchievement struct {
	UserID          string
	AchievementCode string
	GrantedAt       time.Time
	Metadata        json.RawMessage
	SeenAt          *time.Time
	TitleKey        string
	DescriptionKey  string
	Icon            string
	Category        string
}

type UpsertDefinitionInput struct {
	Code           string
	TitleKey       string
	DescriptionKey string
	Icon           string
	Category       string
	CriteriaJSON   json.RawMessage
	IsEnabled      bool
	SortOrder      int
}

type Gateway interface {
	ListEnabledDefinitions() ([]Definition, error)
	ListAllDefinitions() ([]Definition, error)
	GetDefinition(code string) (*Definition, error)
	UpsertDefinition(input UpsertDefinitionInput) (*Definition, error)
	SetDefinitionEnabled(code string, enabled bool) (*Definition, error)

	Grant(userID, code string, metadata json.RawMessage) (granted bool, err error)
	ListUserAchievements(userID string) ([]UserAchievement, error)
	ListUnseen(userID string) ([]UserAchievement, error)
	MarkSeen(userID string, codes []string) error

	RecordLoginDay(userID string, day time.Time) error
	CurrentStreak(userID string) (int, error)

	CountUserProjects(userID string) (int64, error)
	CountUserPublishedProjects(userID string) (int64, error)
	CountUserReactionsGiven(userID string) (int64, error)
	CountReactionsOnOwnedProjects(ownerUserID string) (int64, error)
}
