package router

import "log/slog"

func mustOK(group string, err error) {
	if err != nil {
		slog.Error("router route names check failed", "group", group, "err", err)
		panic("router: " + group + ": " + err.Error())
	}
}

// General – Routen, die keinem fachlichen Modul zugeordnet sind.
var General = struct {
	Index string
	Ping  string
	Panic string
	Info  string
}{
	Index: "general_index",
	Ping:  "general_ping",
	Panic: "general_panic",
	Info:  "general_info",
}

func init() { mustOK("General", EnsureAllFieldsSet(General)) }

// Anon – öffentliche (unauthentifizierte) Routen rund um Login & Passwort.
var Anon = struct {
	LoginPage              string
	Login                  string
	Logoff                 string
	ForgotPasswordPage     string
	ForgotPassword         string
	PasswordCreatePage     string
	SetPasswordByTokenPage string
	SetPasswordByToken     string
	CheckUserToken         string
	Language               string
}{
	LoginPage:              "anon_login_page",
	Login:                  "anon_login",
	Logoff:                 "anon_logoff",
	ForgotPasswordPage:     "anon_forgot_password_page",
	ForgotPassword:         "anon_forgot_password",
	PasswordCreatePage:     "anon_password_create_page",
	SetPasswordByTokenPage: "anon_set_password_by_token_page",
	SetPasswordByToken:     "anon_set_password_by_token",
	CheckUserToken:         "anon_check_user_token",
	Language:               "anon_language",
}

func init() { mustOK("Anon", EnsureAllFieldsSet(Anon)) }

// Auth – authentifizierte Account-Endpoints (Refresh-Token, eigenes Passwort).
var Auth = struct {
	Refresh                   string
	SetPasswordForCurrentUser string
}{
	Refresh:                   "auth_refresh",
	SetPasswordForCurrentUser: "auth_set_password_for_current_user",
}

func init() { mustOK("Auth", EnsureAllFieldsSet(Auth)) }

// Settings – Account-Settings-Page (User-Self-Service).
var Settings = struct {
	Page string
}{
	Page: "settings_page",
}

func init() { mustOK("Settings", EnsureAllFieldsSet(Settings)) }

// Users – Admin-Routen rund um Benutzerverwaltung (Liste/Detail).
var Users = struct {
	List          string
	Detail        string
	Delete        string
	UpdateAccount string
	UpdateGroups  string
}{
	List:          "users_list",
	Detail:        "users_detail",
	Delete:        "users_delete",
	UpdateAccount: "users_update_account",
	UpdateGroups:  "users_update_groups",
}

func init() { mustOK("Users", EnsureAllFieldsSet(Users)) }

// AuthAdmin – Routen aus core/auth/api für die Gruppenverwaltung +
// Admin-User-Endpunkte (UserList, AddUser, Password set/reset durch Admin).
var AuthAdmin = struct {
	GroupList             string
	GroupDetail           string
	GroupUpsert           string
	GroupDelete           string
	GroupAddUser          string
	GroupRemoveUser       string
	GroupAddPermission    string
	GroupRemovePermission string
	UserList              string
	UserAdd               string
}{
	GroupList:             "auth_admin_group_list",
	GroupDetail:           "auth_admin_group_detail",
	GroupUpsert:           "auth_admin_group_upsert",
	GroupDelete:           "auth_admin_group_delete",
	GroupAddUser:          "auth_admin_group_add_user",
	GroupRemoveUser:       "auth_admin_group_remove_user",
	GroupAddPermission:    "auth_admin_group_add_permission",
	GroupRemovePermission: "auth_admin_group_remove_permission",
	UserList:              "auth_admin_user_list",
	UserAdd:               "auth_admin_user_add",
}

func init() { mustOK("AuthAdmin", EnsureAllFieldsSet(AuthAdmin)) }

// AuthUser – Routen aus core/auth/api für angemeldete Benutzer
// (eigene Detaildaten, Passwort setzen/zurücksetzen, Settings-State).
var AuthUser = struct {
	Detail              string
	Update              string
	UpdateSettingsState string
	PasswordSet         string
	PasswordReset       string
}{
	Detail:              "auth_user_detail",
	Update:              "auth_user_update",
	UpdateSettingsState: "auth_user_update_settings_state",
	PasswordSet:         "auth_user_password_set",
	PasswordReset:       "auth_user_password_reset",
}

func init() { mustOK("AuthUser", EnsureAllFieldsSet(AuthUser)) }

// JsLog – Endpoints für das Browser-Frontend zum Reporten von JS-Fehlern.
var JsLog = struct {
	Submit string
	Test   string
	Info   string
}{
	Submit: "jslog_submit",
	Test:   "jslog_test",
	Info:   "jslog_info",
}

func init() { mustOK("JsLog", EnsureAllFieldsSet(JsLog)) }

// Templates – Routen für das HTML-Template-Modul.
var Templates = struct {
	List    string
	New     string
	Detail  string
	Create  string
	Update  string
	Delete  string
	Archive string
}{
	List:    "templates_list",
	New:     "templates_new",
	Detail:  "templates_detail",
	Create:  "templates_create",
	Update:  "templates_update",
	Delete:  "templates_delete",
	Archive: "templates_archive",
}

func init() { mustOK("Templates", EnsureAllFieldsSet(Templates)) }

// Lists – Routen für das Email-Listen-Modul.
var Lists = struct {
	List            string
	New             string
	Detail          string
	Create          string
	Update          string
	Import          string
	Delete          string
	Recipients      string // datatable-Endpoint für Empfänger (GET + POST)
	DeleteRecipient string
	AddRecipient    string
	Archive         string
}{
	List:            "lists_list",
	New:             "lists_new",
	Detail:          "lists_detail",
	Create:          "lists_create",
	Update:          "lists_update",
	Import:          "lists_import",
	Delete:          "lists_delete",
	Recipients:      "lists_recipients",
	DeleteRecipient: "lists_delete_recipient",
	AddRecipient:    "lists_add_recipient",
	Archive:         "lists_archive",
}

func init() { mustOK("Lists", EnsureAllFieldsSet(Lists)) }

// Mailings – Routen für das Versand-Modul.
var Mailings = struct {
	List    string
	New     string
	Detail  string
	Create  string
	Start   string
	Resend  string
	Archive string
}{
	List:    "mailings_list",
	New:     "mailings_new",
	Detail:  "mailings_detail",
	Create:  "mailings_create",
	Start:   "mailings_start",
	Resend:  "mailings_resend",
	Archive: "mailings_archive",
}

func init() { mustOK("Mailings", EnsureAllFieldsSet(Mailings)) }

// Blacklist – globale Sperrliste (gespiegelte Postmark-Suppressions + manuell).
var Blacklist = struct {
	List   string // GET + POST: Seite + datatable Filter/Sort/Page
	Sync   string // POST: Postmark-Suppressions jetzt importieren
	Delete string // DELETE: einzelne Adresse entfernen (reaktiviert in Postmark)
	Export string // GET: komplette Liste als CSV-Download
}{
	List:   "blacklist_list",
	Sync:   "blacklist_sync",
	Delete: "blacklist_delete",
	Export: "blacklist_export",
}

func init() { mustOK("Blacklist", EnsureAllFieldsSet(Blacklist)) }
