package gateway

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type AchievementsGateway struct {
	db *gorm.DB
}

type Module struct {
	fx.Out
	achievements.Gateway
}

func SetupAchievementsGateway(_ db_client.PostgresClient) Module {
	dsn := viper.GetString("projectsPostgres.postgresDsn")
	if dsn == "" {
		panic("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return Module{Gateway: &AchievementsGateway{db: db}}
}

type definitionRow struct {
	Code           string          `gorm:"column:code"`
	TitleKey       string          `gorm:"column:title_key"`
	DescriptionKey string          `gorm:"column:description_key"`
	Icon           string          `gorm:"column:icon"`
	Category       string          `gorm:"column:category"`
	CriteriaJSON   json.RawMessage `gorm:"column:criteria_json"`
	IsEnabled      bool            `gorm:"column:is_enabled"`
	SortOrder      int             `gorm:"column:sort_order"`
	CreatedAt      time.Time       `gorm:"column:created_at"`
	UpdatedAt      time.Time       `gorm:"column:updated_at"`
}

func (r definitionRow) toCore() achievements.Definition {
	return achievements.Definition{
		Code:           r.Code,
		TitleKey:       r.TitleKey,
		DescriptionKey: r.DescriptionKey,
		Icon:           r.Icon,
		Category:       r.Category,
		CriteriaJSON:   r.CriteriaJSON,
		IsEnabled:      r.IsEnabled,
		SortOrder:      r.SortOrder,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func (g *AchievementsGateway) ListEnabledDefinitions() ([]achievements.Definition, error) {
	var rows []definitionRow
	err := g.db.Raw(`
		SELECT code, title_key, description_key, icon, category, criteria_json, is_enabled, sort_order, created_at, updated_at
		FROM achievement_definitions
		WHERE is_enabled = TRUE
		ORDER BY sort_order ASC, code ASC
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]achievements.Definition, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toCore())
	}
	return out, nil
}

func (g *AchievementsGateway) ListAllDefinitions() ([]achievements.Definition, error) {
	var rows []definitionRow
	err := g.db.Raw(`
		SELECT code, title_key, description_key, icon, category, criteria_json, is_enabled, sort_order, created_at, updated_at
		FROM achievement_definitions
		ORDER BY sort_order ASC, code ASC
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]achievements.Definition, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toCore())
	}
	return out, nil
}

func (g *AchievementsGateway) GetDefinition(code string) (*achievements.Definition, error) {
	var row definitionRow
	err := g.db.Raw(`
		SELECT code, title_key, description_key, icon, category, criteria_json, is_enabled, sort_order, created_at, updated_at
		FROM achievement_definitions WHERE code = ?
	`, code).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.Code == "" {
		return nil, gorm.ErrRecordNotFound
	}
	core := row.toCore()
	return &core, nil
}

func (g *AchievementsGateway) UpsertDefinition(input achievements.UpsertDefinitionInput) (*achievements.Definition, error) {
	criteria := input.CriteriaJSON
	if len(criteria) == 0 {
		criteria = json.RawMessage(`{}`)
	}
	err := g.db.Exec(`
		INSERT INTO achievement_definitions (
			code, title_key, description_key, icon, category, criteria_json, is_enabled, sort_order, updated_at
		) VALUES (?, ?, ?, ?, ?, ?::jsonb, ?, ?, now())
		ON CONFLICT (code) DO UPDATE SET
			title_key = EXCLUDED.title_key,
			description_key = EXCLUDED.description_key,
			icon = EXCLUDED.icon,
			category = EXCLUDED.category,
			criteria_json = EXCLUDED.criteria_json,
			is_enabled = EXCLUDED.is_enabled,
			sort_order = EXCLUDED.sort_order,
			updated_at = now()
	`,
		strings.TrimSpace(input.Code),
		strings.TrimSpace(input.TitleKey),
		strings.TrimSpace(input.DescriptionKey),
		strings.TrimSpace(input.Icon),
		strings.TrimSpace(input.Category),
		string(criteria),
		input.IsEnabled,
		input.SortOrder,
	).Error
	if err != nil {
		return nil, err
	}
	return g.GetDefinition(strings.TrimSpace(input.Code))
}

func (g *AchievementsGateway) SetDefinitionEnabled(code string, enabled bool) (*achievements.Definition, error) {
	err := g.db.Exec(`
		UPDATE achievement_definitions SET is_enabled = ?, updated_at = now() WHERE code = ?
	`, enabled, code).Error
	if err != nil {
		return nil, err
	}
	return g.GetDefinition(code)
}

func (g *AchievementsGateway) Grant(userID, code string, metadata json.RawMessage) (bool, error) {
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	res := g.db.Exec(`
		INSERT INTO user_achievements (user_id, achievement_code, metadata)
		VALUES (?, ?, ?::jsonb)
		ON CONFLICT (user_id, achievement_code) DO NOTHING
	`, userID, code, string(metadata))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

type userAchievementRow struct {
	UserID          string          `gorm:"column:user_id"`
	AchievementCode string          `gorm:"column:achievement_code"`
	GrantedAt       time.Time       `gorm:"column:granted_at"`
	Metadata        json.RawMessage `gorm:"column:metadata"`
	SeenAt          *time.Time      `gorm:"column:seen_at"`
	TitleKey        string          `gorm:"column:title_key"`
	DescriptionKey  string          `gorm:"column:description_key"`
	Icon            string          `gorm:"column:icon"`
	Category        string          `gorm:"column:category"`
}

func (g *AchievementsGateway) ListUserAchievements(userID string) ([]achievements.UserAchievement, error) {
	var rows []userAchievementRow
	err := g.db.Raw(`
		SELECT ua.user_id, ua.achievement_code, ua.granted_at, ua.metadata, ua.seen_at,
		       d.title_key, d.description_key, d.icon, d.category
		FROM user_achievements ua
		JOIN achievement_definitions d ON d.code = ua.achievement_code
		WHERE ua.user_id = ?
		ORDER BY ua.granted_at DESC
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]achievements.UserAchievement, 0, len(rows))
	for _, row := range rows {
		out = append(out, achievements.UserAchievement{
			UserID:          row.UserID,
			AchievementCode: row.AchievementCode,
			GrantedAt:       row.GrantedAt,
			Metadata:        row.Metadata,
			SeenAt:          row.SeenAt,
			TitleKey:        row.TitleKey,
			DescriptionKey:  row.DescriptionKey,
			Icon:            row.Icon,
			Category:        row.Category,
		})
	}
	return out, nil
}

func (g *AchievementsGateway) ListUnseen(userID string) ([]achievements.UserAchievement, error) {
	var rows []userAchievementRow
	err := g.db.Raw(`
		SELECT ua.user_id, ua.achievement_code, ua.granted_at, ua.metadata, ua.seen_at,
		       d.title_key, d.description_key, d.icon, d.category
		FROM user_achievements ua
		JOIN achievement_definitions d ON d.code = ua.achievement_code
		WHERE ua.user_id = ? AND ua.seen_at IS NULL
		ORDER BY ua.granted_at ASC
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]achievements.UserAchievement, 0, len(rows))
	for _, row := range rows {
		out = append(out, achievements.UserAchievement{
			UserID:          row.UserID,
			AchievementCode: row.AchievementCode,
			GrantedAt:       row.GrantedAt,
			Metadata:        row.Metadata,
			SeenAt:          row.SeenAt,
			TitleKey:        row.TitleKey,
			DescriptionKey:  row.DescriptionKey,
			Icon:            row.Icon,
			Category:        row.Category,
		})
	}
	return out, nil
}

func (g *AchievementsGateway) MarkSeen(userID string, codes []string) error {
	if len(codes) == 0 {
		return g.db.Exec(`
			UPDATE user_achievements SET seen_at = now()
			WHERE user_id = ? AND seen_at IS NULL
		`, userID).Error
	}
	return g.db.Exec(`
		UPDATE user_achievements SET seen_at = now()
		WHERE user_id = ? AND seen_at IS NULL AND achievement_code IN ?
	`, userID, codes).Error
}

func (g *AchievementsGateway) RecordLoginDay(userID string, day time.Time) error {
	d := day.UTC().Format("2006-01-02")
	return g.db.Exec(`
		INSERT INTO user_login_days (user_id, login_date)
		VALUES (?, ?::date)
		ON CONFLICT (user_id, login_date) DO NOTHING
	`, userID, d).Error
}

func (g *AchievementsGateway) CurrentStreak(userID string) (int, error) {
	var dates []time.Time
	err := g.db.Raw(`
		SELECT login_date FROM user_login_days
		WHERE user_id = ?
		ORDER BY login_date DESC
		LIMIT 60
	`, userID).Scan(&dates).Error
	if err != nil {
		return 0, err
	}
	if len(dates) == 0 {
		return 0, nil
	}
	streak := 0
	var expected time.Time
	for i, d := range dates {
		day := d.UTC().Truncate(24 * time.Hour)
		if i == 0 {
			streak = 1
			expected = day.Add(-24 * time.Hour)
			continue
		}
		if day.Equal(expected) {
			streak++
			expected = expected.Add(-24 * time.Hour)
			continue
		}
		break
	}
	return streak, nil
}

func (g *AchievementsGateway) CountUserProjects(userID string) (int64, error) {
	var count int64
	err := g.db.Raw(`SELECT COUNT(*) FROM scratch_projects WHERE owner_user_id = ? AND deleted_at IS NULL`, userID).Scan(&count).Error
	return count, err
}

func (g *AchievementsGateway) CountUserPublishedProjects(userID string) (int64, error) {
	var count int64
	err := g.db.Raw(`
		SELECT COUNT(*) FROM scratch_projects
		WHERE owner_user_id = ? AND is_public = TRUE AND deleted_at IS NULL
	`, userID).Scan(&count).Error
	return count, err
}

func (g *AchievementsGateway) CountUserReactionsGiven(userID string) (int64, error) {
	var count int64
	err := g.db.Raw(`SELECT COUNT(*) FROM scratch_project_reactions WHERE user_id = ?`, userID).Scan(&count).Error
	return count, err
}

func (g *AchievementsGateway) CountReactionsOnOwnedProjects(ownerUserID string) (int64, error) {
	var count int64
	err := g.db.Raw(`
		SELECT COUNT(*) FROM scratch_project_reactions r
		JOIN scratch_projects p ON p.id = r.project_id
		WHERE p.owner_user_id = ? AND p.deleted_at IS NULL
	`, ownerUserID).Scan(&count).Error
	return count, err
}
