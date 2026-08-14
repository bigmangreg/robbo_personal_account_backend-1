package resolvers

import (
	"log"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
)

func (r *Resolver) safeEvaluateProfile(userID, fullName string) {
	if r.achievementsDelegate == nil {
		return
	}
	if err := r.achievementsDelegate.Evaluate(userID, achievements.EventProfileUpdate, achievements.EvaluatePayload{
		FullName: fullName,
	}); err != nil {
		log.Printf("achievements: profile evaluate: %v", err)
	}
}
