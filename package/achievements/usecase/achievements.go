package usecase

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
	"go.uber.org/fx"
)

type AchievementsUseCaseImpl struct {
	gateway achievements.Gateway
}

type Module struct {
	fx.Out
	achievements.UseCase
}

func SetupAchievementsUseCase(gateway achievements.Gateway) Module {
	return Module{UseCase: &AchievementsUseCaseImpl{gateway: gateway}}
}

type criteria struct {
	Event           string `json:"event"`
	MinCount        int64  `json:"min_count"`
	MinStreak       int    `json:"min_streak"`
	RequireFullName bool   `json:"require_full_name"`
}

func (u *AchievementsUseCaseImpl) Evaluate(userID, event string, payload achievements.EvaluatePayload) error {
	userID = strings.TrimSpace(userID)
	event = strings.TrimSpace(event)
	if userID == "" || event == "" {
		return nil
	}

	if event == achievements.EventLogin {
		if err := u.gateway.RecordLoginDay(userID, time.Now().UTC()); err != nil {
			log.Printf("achievements: record login day: %v", err)
		}
	}

	defs, err := u.gateway.ListEnabledDefinitions()
	if err != nil {
		return err
	}

	for _, def := range defs {
		var c criteria
		if len(def.CriteriaJSON) > 0 {
			_ = json.Unmarshal(def.CriteriaJSON, &c)
		}
		if c.Event == "" {
			c.Event = event
		}
		if c.Event != event {
			continue
		}
		ok, evalErr := u.matches(userID, event, payload, c)
		if evalErr != nil {
			log.Printf("achievements: evaluate %s for %s: %v", def.Code, userID, evalErr)
			continue
		}
		if !ok {
			continue
		}
		meta, _ := json.Marshal(map[string]interface{}{
			"event": event,
		})
		if _, grantErr := u.gateway.Grant(userID, def.Code, meta); grantErr != nil {
			log.Printf("achievements: grant %s to %s: %v", def.Code, userID, grantErr)
		}
	}

	// reaction_received evaluates for project owner separately
	if event == achievements.EventReactionPut {
		ownerID := strings.TrimSpace(payload.TargetUserID)
		if ownerID != "" && ownerID != userID {
			_ = u.Evaluate(ownerID, achievements.EventReactionReceived, achievements.EvaluatePayload{})
		}
	}
	return nil
}

func (u *AchievementsUseCaseImpl) matches(
	userID, event string,
	payload achievements.EvaluatePayload,
	c criteria,
) (bool, error) {
	minCount := c.MinCount
	if minCount <= 0 {
		minCount = 1
	}

	switch event {
	case achievements.EventLogin:
		if c.MinStreak > 0 {
			streak, err := u.gateway.CurrentStreak(userID)
			if err != nil {
				return false, err
			}
			return streak >= c.MinStreak, nil
		}
		return true, nil
	case achievements.EventProfileUpdate:
		if c.RequireFullName {
			return strings.TrimSpace(payload.FullName) != "", nil
		}
		return true, nil
	case achievements.EventProjectCreate:
		n, err := u.gateway.CountUserProjects(userID)
		if err != nil {
			return false, err
		}
		return n >= minCount, nil
	case achievements.EventProjectPublish:
		n, err := u.gateway.CountUserPublishedProjects(userID)
		if err != nil {
			return false, err
		}
		return n >= minCount, nil
	case achievements.EventReactionPut:
		n, err := u.gateway.CountUserReactionsGiven(userID)
		if err != nil {
			return false, err
		}
		return n >= minCount, nil
	case achievements.EventReactionReceived:
		n, err := u.gateway.CountReactionsOnOwnedProjects(userID)
		if err != nil {
			return false, err
		}
		return n >= minCount, nil
	default:
		return false, nil
	}
}

func (u *AchievementsUseCaseImpl) ListForUser(userID string) (*achievements.UserAchievementsView, error) {
	userID = strings.TrimSpace(userID)
	defs, err := u.gateway.ListEnabledDefinitions()
	if err != nil {
		return nil, err
	}
	earnedRows, err := u.gateway.ListUserAchievements(userID)
	if err != nil {
		return nil, err
	}
	earnedMap := map[string]achievements.UserAchievement{}
	for _, row := range earnedRows {
		earnedMap[row.AchievementCode] = row
	}

	view := &achievements.UserAchievementsView{
		UserID:    userID,
		Earned:    []achievements.AchievementView{},
		Available: []achievements.AchievementView{},
	}
	for _, def := range defs {
		item := achievements.AchievementView{
			Code:           def.Code,
			TitleKey:       def.TitleKey,
			DescriptionKey: def.DescriptionKey,
			Icon:           def.Icon,
			Category:       def.Category,
			IsEarned:       false,
		}
		if row, ok := earnedMap[def.Code]; ok {
			granted := row.GrantedAt
			item.IsEarned = true
			item.GrantedAt = &granted
			view.Earned = append(view.Earned, item)
		} else {
			view.Available = append(view.Available, item)
		}
	}
	return view, nil
}

func (u *AchievementsUseCaseImpl) ListUnseen(userID string) ([]achievements.UserAchievement, error) {
	return u.gateway.ListUnseen(strings.TrimSpace(userID))
}

func (u *AchievementsUseCaseImpl) MarkSeen(userID string, codes []string) error {
	return u.gateway.MarkSeen(strings.TrimSpace(userID), codes)
}

func (u *AchievementsUseCaseImpl) ListDefinitions() ([]achievements.Definition, error) {
	return u.gateway.ListAllDefinitions()
}

func (u *AchievementsUseCaseImpl) UpsertDefinition(input achievements.UpsertDefinitionInput) (*achievements.Definition, error) {
	input.Code = strings.TrimSpace(input.Code)
	input.TitleKey = strings.TrimSpace(input.TitleKey)
	input.DescriptionKey = strings.TrimSpace(input.DescriptionKey)
	if input.Icon == "" {
		input.Icon = "🏆"
	}
	if input.Category == "" {
		input.Category = "general"
	}
	if len(input.CriteriaJSON) == 0 {
		input.CriteriaJSON = json.RawMessage(`{}`)
	}
	return u.gateway.UpsertDefinition(input)
}

func (u *AchievementsUseCaseImpl) SetDefinitionEnabled(code string, enabled bool) (*achievements.Definition, error) {
	return u.gateway.SetDefinitionEnabled(strings.TrimSpace(code), enabled)
}
