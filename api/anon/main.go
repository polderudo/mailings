package anon

import (
	anonauth "app/api/anon/auth"
	anonpages "app/api/anon/pages"
	"app/api/router"
	"app/mw"
	"net/http"
)

func CreateRoutes(rAnon *mw.AnonGroup, rAuth *mw.UserGroup) {
	// Index / Misc
	rAnon.AddNamedAnonRoute(router.General.Index, "/", anonpages.Index, http.MethodGet)
	rAnon.AddNamedRoute(router.General.Ping, "/ping/", Ping, http.MethodGet)
	rAnon.AddNamedRoute(router.General.Panic, "/panic/", Panic, http.MethodGet)
	rAnon.AddNamedRoute(router.General.Info, "/info/", Info, http.MethodGet)

	// Auth (anon)
	rAnon.AddNamedAnonRoute(router.Anon.LoginPage, "/auth/login/", anonpages.LoginPage, http.MethodGet)
	rAnon.AddNamedRoute(router.Anon.Login, "/auth/login/", anonauth.Login, http.MethodPost)
	rAnon.AddNamedRoute(router.Anon.Logoff, "/auth/logoff/", anonauth.Logoff, http.MethodPost)
	rAnon.AddNamedAnonRoute(router.Anon.ForgotPasswordPage, "/auth/forgot_password/", anonpages.ForgotPasswordPage, http.MethodGet)
	rAnon.AddNamedRoute(router.Anon.ForgotPassword, "/auth/forgot_password/", anonauth.ForgotPassword, http.MethodPost)
	rAnon.AddNamedAnonRoute(router.Anon.PasswordCreatePage, "/password_create/:token/", anonpages.PasswordCreatePage, http.MethodGet)
	rAnon.AddNamedRoute(router.Anon.SetPasswordByTokenPage, "/password_create/:token/", anonauth.SetPasswordByTokenPage, http.MethodPost)
	rAnon.AddNamedRoute(router.Anon.SetPasswordByToken, "/auth/set_password_by_token/", anonauth.SetPasswordByToken, http.MethodPost)
	rAnon.AddNamedRoute(router.Anon.CheckUserToken, "/check_user_token/", anonauth.CheckUserToken, http.MethodGet)
	rAnon.AddNamedRoute(router.Anon.Language, "/language/:lang/", anonauth.SetLanguage, http.MethodGet)

	// Settings / Users (anon-Page-Gate, intern wird auf HasUser geprüft)
	rAnon.AddNamedAnonRoute(router.Settings.Page, "/settings/", anonpages.SettingsPage, http.MethodGet)
	// GET + POST: GET = initiale Page, POST = datatable Filter/Sort/Page-Submit.
	rAnon.AddNamedAnonRoute(router.Users.List, "/users/", anonpages.UsersPage, http.MethodGet, http.MethodPost)
	rAnon.AddNamedAnonRoute(router.Users.Detail, "/users/:id/", anonpages.UserDetailPage, http.MethodGet)

	// Auth (eingeloggt)
	rAuth.AddNamedRoute(router.Auth.Refresh, "/auth/refresh/", anonauth.Refresh, http.MethodPost)
	rAuth.AddNamedRoute(router.Auth.SetPasswordForCurrentUser, "/auth/set_password/", anonauth.SetPasswordForCurrentUser, http.MethodPost)

	// Admin-actions auf User-Datensätzen
	rAuth.AddNamedRoute(router.Users.Delete, "/users/:id/", anonpages.UserDelete, http.MethodDelete)
	rAuth.AddNamedRoute(router.Users.UpdateAccount, "/users/:id/account/", anonpages.UserUpdateAccount, http.MethodPost)
	rAuth.AddNamedRoute(router.Users.UpdateGroups, "/users/:id/groups/", anonpages.UserUpdateGroups, http.MethodPost)
}
