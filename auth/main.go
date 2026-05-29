package auth

import (
	"app/mq"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	"github.com/aarondl/opt/null"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/stephenafamo/bob"
)

var (
	enforcer *casbin.Enforcer
)

func InitAuth(db *sql.DB, rbacModel string) error {
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		panic(err)
	}

	// Create the enforcer.
	a, err := sqladapter.NewAdapter(db, "postgres", "casbin_rules_v1")
	if err != nil {
		return fmt.Errorf("create casbin adapter: %w", err)
	}

	enforcer, err = casbin.NewEnforcer(m, a)
	if err != nil {
		return fmt.Errorf("create casbin enforcer: %w", err)
	}

	// Load the policy from DB.
	if err = enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("load casbin policy: %w", err)
	}

	if err := InitPolicies(); err != nil {
		return fmt.Errorf("init policies.go: %w", err)
	}

	return nil
}

func GetUserGroups(userID int32) []string {
	roles, err := enforcer.GetRolesForUser(fmt.Sprintf("%d", userID))
	if err != nil {
		log.Println("get roles for user:", err)
		return []string{}
	}

	return roles
}
func AddGroupPermission(group string, permission string, action string) error {
	if _, err := enforcer.AddPolicy(group, permission, action); err != nil {
		return fmt.Errorf("error add group policy: %w", err)
	}
	return nil
}

func RemoveGroupPermission(group string, permission string, action string) error {
	if _, err := enforcer.RemovePolicy(group, permission, action); err != nil {
		return fmt.Errorf("error add group policy: %w", err)
	}
	return nil
}

func GetGroupPermissions(groupID string) []string {
	pms, err := enforcer.GetFilteredPolicy(0, groupID)
	if err != nil {
		log.Println("get permissions for group:", err)
		return []string{}
	}

	permissions := []string{}
	for _, p := range pms {
		permissions = append(permissions, p[1])
	}

	return permissions
}

func GetUserPermissions(userID int32) []string {
	if IsAdmin(userID) {
		return []string{OpAll}
	} else if IsSuperAdmin(userID) {
		return []string{OpAllSuper}
	}

	pms, err := enforcer.GetImplicitPermissionsForUser(fmt.Sprintf("%d", userID))
	if err != nil {
		log.Println("get permissions for user:", err)
		return []string{}
	}

	permissions := []string{}
	for _, p := range pms {
		permissions = append(permissions, p[1])
	}

	return permissions
}

func GetAllUsersForGroup(group string) ([]int32, error) {
	u, err := enforcer.GetUsersForRole(group, "")
	if err != nil {
		return nil, fmt.Errorf("error reading group users: %w", err)
	}

	var ret []int32
	for i := 0; i < len(u); i++ {
		ui, err := strconv.Atoi(u[i])
		if err != nil {
			return nil, fmt.Errorf("error converting user id %s to int: %w", u[i], err)
		}
		ret = append(ret, int32(ui))
	}

	return ret, nil
}

type UserProfile struct {
	ID                     int32               `db:"id,pk" json:"id"`
	Email                  string              `db:"email" json:"email"`
	Salutation             string              `db:"salutation" json:"salutation"`
	Forename               string              `db:"forename" json:"forename"`
	Lastname               string              `db:"lastname" json:"lastname"`
	MustResetPassword      bool                `db:"must_reset_password" json:"must_reset_password"`
	LastLogin              null.Val[time.Time] `db:"last_login" json:"last_login"`
	CreatedByUserProfileID null.Val[int32]     `db:"created_by_user_profile_id" json:"created_by_user_profile_id"`
	Disabled               bool                `db:"disabled" json:"disabled"`
	EmailValidated         bool                `db:"email_validated" json:"email_validated"`
	CreatedAt              time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt              null.Val[time.Time] `db:"updated_at" json:"updated_at"`
	UserGroups             []string            `db:"-" json:"user_groups"`
	Permissions            []string            `db:"-" json:"permissions"`
}

func GetUserDetails(ctx context.Context, tx bob.Executor, userID int32, withGroups bool, withPermissions bool) *UserProfile {
	user, err := mq.FindUserProfile(ctx, tx, userID)
	var u *UserProfile
	if err == nil && user != nil {
		u = &UserProfile{
			ID:                     user.ID,
			Email:                  user.Email,
			Salutation:             user.Salutation,
			Forename:               user.Forename,
			Lastname:               user.Lastname,
			MustResetPassword:      user.MustResetPassword,
			LastLogin:              user.LastLogin,
			CreatedByUserProfileID: user.CreatedByUserProfileID,
			Disabled:               user.Disabled,
			EmailValidated:         user.EmailValidated,
			CreatedAt:              user.CreatedAt,
			UpdatedAt:              user.UpdatedAt,
		}
		if withGroups {
			u.UserGroups = GetUserGroups(userID)
		}
		if withPermissions {
			u.Permissions = GetUserPermissions(userID)
		}
	} else {
		log.Println("error selecting user in details:", err)
	}

	return u
}

type UserDescription struct {
	UserID int32  `json:"user_id"`
	Name   string `json:"name"`
}

func GetUserDescription(ctx context.Context, tx bob.Executor, userID int32, user *mq.UserProfile) *UserDescription {
	var err error
	if user == nil {
		user, err = mq.FindUserProfile(ctx, tx, userID)
	}

	if err == nil && user != nil {
		return &UserDescription{
			UserID: user.ID,
			Name:   user.Forename + " " + user.Lastname + " (" + user.Email + ")",
		}
	} else {
		log.Println("error selecting user in details:", err)
	}

	return &UserDescription{
		UserID: 0,
		Name:   "-NA-",
	}
}

func GetAllUsedGroups() []string {
	roles, err := enforcer.GetAllRoles()
	if err != nil {
		log.Println("error get roles for group:", err)
	}
	return roles
}

func isUserInGroup(userID int32, group string) bool {
	if group == "" {
		return false
	}

	has, err := enforcer.HasRoleForUser(fmt.Sprintf("%d", userID), group)
	if err != nil {
		log.Println("check user role error:", err)
		return false
	}
	return has
}

func AddUserToGroup(userID int32, group string) error {
	if group != "" {
		if _, err := enforcer.AddRoleForUser(fmt.Sprintf("%d", userID), group); err != nil {
			return fmt.Errorf("error add role for user: %w", err)
		}
	}

	return nil
}

func RemoveUserFromGroup(userID int32, group string) error {
	if group != "" {
		if _, err := enforcer.DeleteRoleForUser(fmt.Sprintf("%d", userID), group); err != nil {
			return fmt.Errorf("error removing role from user: %w", err)
		}
	}

	return nil
}
