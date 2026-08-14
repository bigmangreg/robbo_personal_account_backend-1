package resolvers

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/achievements"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/courses"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboGroup"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/robboUnits"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/users"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	authDelegate         auth.Delegate
	usersDelegate        users.Delegate
	robboGroupDelegate   robboGroup.Delegate
	robboUnitsDelegate   robboUnits.Delegate
	coursesDelegate      courses.Delegate
	projectPageDelegate  projectPage.Delegate
	achievementsDelegate achievements.Delegate
}

type MutationResolver struct{ *Resolver }
type QueryResolver struct{ *Resolver }

func GinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: "internal server error",
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		return nil, &gqlerror.Error{
			Path:    graphql.GetPath(ctx),
			Message: "gin.Context has wrong type",
			Extensions: map[string]interface{}{
				"code": "500",
			},
		}
	}
	return gc, nil
}

func NewResolver(
	authDelegate auth.Delegate,
	usersDelegate users.Delegate,
	robboGroupDelegate robboGroup.Delegate,
	robboUnitsDelegate robboUnits.Delegate,
	coursesDelegate courses.Delegate,
	projectPageDelegate projectPage.Delegate,
	achievementsDelegate achievements.Delegate,
) Resolver {
	return Resolver{
		authDelegate:         authDelegate,
		usersDelegate:        usersDelegate,
		robboGroupDelegate:   robboGroupDelegate,
		robboUnitsDelegate:   robboUnitsDelegate,
		coursesDelegate:      coursesDelegate,
		projectPageDelegate:  projectPageDelegate,
		achievementsDelegate: achievementsDelegate,
	}
}
