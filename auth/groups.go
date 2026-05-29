package auth

const (
	GroupSuperAdmin = "super_admin"
	GroupDeveloper  = "developer"
	GroupAdmin      = "admin"
	GroupOperator   = "operator"
	GroupOffice     = "office"
)

func IsSuperAdmin(userID int32) bool {
	return isUserInGroup(userID, GroupSuperAdmin)
}

func IsAdmin(userID int32) bool {
	return isUserInGroup(userID, GroupAdmin) || IsSuperAdmin(userID)
}

func IsOperator(userID int32) bool {
	return isUserInGroup(userID, GroupOperator) || IsAdmin(userID)
}

func IsOffice(userID int32) bool {
	return isUserInGroup(userID, GroupOffice) || IsAdmin(userID)
}

func IsDeveloper(userID int32) bool {
	return isUserInGroup(userID, GroupDeveloper)
}
