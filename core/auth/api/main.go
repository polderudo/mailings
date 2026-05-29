package api

import (
	"app/api/router"
	"app/mw"
	"context"
	"net/http"
)

type api struct {
	DbCtx context.Context
}

var (
	authApi = &api{
		DbCtx: context.Background(),
	}
)

func CreateRoutes(rUser *mw.UserGroup, rAdmin *mw.UserGroup) {
	// Eigener User
	rUser.AddNamedRoute(router.AuthUser.Detail, "/auth/user/detail/", authApi.UserDetails, http.MethodGet)
	rUser.AddNamedRoute(router.AuthUser.Update, "/auth/user/detail/", authApi.UserUpdate, http.MethodPost)
	rUser.AddNamedRoute(router.AuthUser.UpdateSettingsState, "/auth/user/settings/detail/", authApi.UserUpdateSettingsState, http.MethodPost)
	rUser.AddNamedRoute(router.AuthUser.PasswordSet, "/auth/user/password/set/", authApi.SetUserPassword, http.MethodPost)
	rUser.AddNamedRoute(router.AuthUser.PasswordReset, "/auth/user/password/reset/", authApi.ResetUserPassword, http.MethodPost)

	// Admin: User-Verwaltung
	rAdmin.AddNamedRoute(router.AuthAdmin.UserList, "/auth/user/list/", authApi.UserList, http.MethodGet)
	rAdmin.AddNamedRoute(router.AuthAdmin.UserAdd, "/auth/user/add/", authApi.AddUser, http.MethodPost)

	// Admin: Gruppen
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupList, "/auth/group/list/", authApi.ListGroups, http.MethodGet)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupDetail, "/auth/group/detail/", authApi.GroupDetails, http.MethodGet)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupUpsert, "/auth/group/upsert/", authApi.GroupUpsert, http.MethodPost)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupDelete, "/auth/group/delete/", authApi.GroupDelete, http.MethodPost)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupAddUser, "/auth/group/add/", authApi.AddUserToGroup, http.MethodPost)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupRemoveUser, "/auth/group/remove/", authApi.RemoveUserFromGroup, http.MethodPost)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupAddPermission, "/auth/group/add_permission/", authApi.AddGroupPermission, http.MethodPost)
	rAdmin.AddNamedRoute(router.AuthAdmin.GroupRemovePermission, "/auth/group/remove_permission/", authApi.RemoveGroupPermission, http.MethodPost)
}
