package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type Handler struct {
	authDelegate auth.Delegate
	useCase      achievements.UseCase
}

func NewAchievementsHandler(authDelegate auth.Delegate, useCase achievements.UseCase) Handler {
	return Handler{authDelegate: authDelegate, useCase: useCase}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	group := router.Group("/api/achievements")
	group.GET("/unseen", h.Unseen)
	group.POST("/mark-seen", h.MarkSeen)
	group.GET("/user/:userId", h.ListForUser)
	group.GET("/definitions", h.ListDefinitions)
	group.PUT("/definitions", h.UpsertDefinition)
	group.POST("/definitions/:code/enabled", h.SetDefinitionEnabled)
}

func allRoles() []models.Role {
	return []models.Role{
		models.Student,
		models.Teacher,
		models.Parent,
		models.FreeListener,
		models.UnitAdmin,
		models.SuperAdmin,
	}
}

func (h *Handler) identity(c *gin.Context, roles []models.Role) (string, models.Role, bool) {
	userID, role, err := h.authDelegate.UserIdentity(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", 0, false
	}
	if err := h.authDelegate.UserAccess(role, roles, c); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return "", 0, false
	}
	return userID, role, true
}

func achievementJSON(v achievements.AchievementView) gin.H {
	out := gin.H{
		"code":           v.Code,
		"titleKey":       v.TitleKey,
		"descriptionKey": v.DescriptionKey,
		"icon":           v.Icon,
		"category":       v.Category,
		"isEarned":       v.IsEarned,
	}
	if v.GrantedAt != nil {
		out["grantedAt"] = v.GrantedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

func definitionJSON(d achievements.Definition) gin.H {
	criteria := string(d.CriteriaJSON)
	if criteria == "" {
		criteria = "{}"
	}
	return gin.H{
		"code":           d.Code,
		"titleKey":       d.TitleKey,
		"descriptionKey": d.DescriptionKey,
		"icon":           d.Icon,
		"category":       d.Category,
		"criteriaJson":   criteria,
		"isEnabled":      d.IsEnabled,
		"sortOrder":      d.SortOrder,
	}
}

func (h *Handler) Unseen(c *gin.Context) {
	userID, _, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	items, err := h.useCase.ListUnseen(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{
			"code":           item.AchievementCode,
			"titleKey":       item.TitleKey,
			"descriptionKey": item.DescriptionKey,
			"icon":           item.Icon,
			"category":       item.Category,
			"grantedAt":      item.GrantedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

type markSeenInput struct {
	Codes []string `json:"codes"`
}

func (h *Handler) MarkSeen(c *gin.Context) {
	userID, _, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	var input markSeenInput
	_ = c.ShouldBindJSON(&input)
	if err := h.useCase.MarkSeen(userID, input.Codes); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListForUser(c *gin.Context) {
	_, _, ok := h.identity(c, allRoles())
	if !ok {
		return
	}
	targetID := strings.TrimSpace(c.Param("userId"))
	if targetID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}
	view, err := h.useCase.ListForUser(targetID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	earned := make([]gin.H, 0, len(view.Earned))
	available := make([]gin.H, 0, len(view.Available))
	for _, item := range view.Earned {
		earned = append(earned, achievementJSON(item))
	}
	for _, item := range view.Available {
		available = append(available, achievementJSON(item))
	}
	c.JSON(http.StatusOK, gin.H{
		"userId":    view.UserID,
		"earned":    earned,
		"available": available,
	})
}

func (h *Handler) ListDefinitions(c *gin.Context) {
	_, _, ok := h.identity(c, []models.Role{models.SuperAdmin})
	if !ok {
		return
	}
	defs, err := h.useCase.ListDefinitions()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(defs))
	for _, d := range defs {
		out = append(out, definitionJSON(d))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

type upsertDefinitionBody struct {
	Code           string `json:"code"`
	TitleKey       string `json:"titleKey"`
	DescriptionKey string `json:"descriptionKey"`
	Icon           string `json:"icon"`
	Category       string `json:"category"`
	CriteriaJSON   string `json:"criteriaJson"`
	IsEnabled      bool   `json:"isEnabled"`
	SortOrder      int    `json:"sortOrder"`
}

func (h *Handler) UpsertDefinition(c *gin.Context) {
	_, _, ok := h.identity(c, []models.Role{models.SuperAdmin})
	if !ok {
		return
	}
	var body upsertDefinitionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	criteria := json.RawMessage([]byte(strings.TrimSpace(body.CriteriaJSON)))
	if !json.Valid(criteria) {
		criteria = json.RawMessage(`{}`)
	}
	def, err := h.useCase.UpsertDefinition(achievements.UpsertDefinitionInput{
		Code:           body.Code,
		TitleKey:       body.TitleKey,
		DescriptionKey: body.DescriptionKey,
		Icon:           body.Icon,
		Category:       body.Category,
		CriteriaJSON:   criteria,
		IsEnabled:      body.IsEnabled,
		SortOrder:      body.SortOrder,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, definitionJSON(*def))
}

type setEnabledBody struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) SetDefinitionEnabled(c *gin.Context) {
	_, _, ok := h.identity(c, []models.Role{models.SuperAdmin})
	if !ok {
		return
	}
	var body setEnabledBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	def, err := h.useCase.SetDefinitionEnabled(c.Param("code"), body.Enabled)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, definitionJSON(*def))
}
