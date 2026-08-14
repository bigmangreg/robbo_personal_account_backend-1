package delegate

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
	"go.uber.org/fx"
)

type AchievementsDelegateImpl struct {
	achievements.UseCase
}

type Module struct {
	fx.Out
	achievements.Delegate
}

func SetupAchievementsDelegate(useCase achievements.UseCase) Module {
	return Module{Delegate: &AchievementsDelegateImpl{UseCase: useCase}}
}
