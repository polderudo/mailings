package mail

import (
	"app/conf"
	"fmt"
)

func GetSubject(subject string) string {
	if conf.C.IsDevSystem || conf.C.ApplicationName == "MESA (Stage)" {
		return fmt.Sprintf("%s [%s]", subject, conf.C.ApplicationName)
	}
	return subject
}
