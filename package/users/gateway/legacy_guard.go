package gateway

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

var ErrLegacyUsersDisabled = errors.New("legacy user store disabled: use OIDC sign-in")

// legacyUsersEnabled controls whether users/profile CRUD uses legacy robbo_db (*_dbs).
// legacyPostgres.enabled alone is not enough: after cutover, auth identities live in LMS MySQL
// (auth_user). Enabling legacy PG for units/groups must not break GetUser/profile for OIDC/LMS logins.
func legacyUsersEnabled() bool {
	if !viper.GetBool("legacyPostgres.enabled") {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(viper.GetString("auth.mode")))
	if mode == "oidc_bff" || mode == "lms_db" {
		return false
	}
	return true
}
