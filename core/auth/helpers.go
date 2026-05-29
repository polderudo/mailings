package auth

import "strings"

// TrimmedLower strips all blanks and returns and also lowerCases the email
func TrimmedLower(email string) string {
	email = strings.ToLower(email)
	email = strings.Trim(email, "\r\n\t ")
	return email
}
