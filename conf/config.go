package conf

type Config struct {
	IsDevSystem       bool
	BindLocalhost     bool
	ShowDebugWindow   bool
	MailToDevelop     string
	ApplicationAdmins []string
	SystemLangISOCode string

	LogWebToConsole bool
	LogWebToDebug   bool

	UseHTTPs      bool
	HTTPSKeyFile  string
	HTTPSCertFile string

	Logging Logging

	Port                    int64
	JWTSecreteKey           string
	JWTTokenDurationMinutes int
	SensibleDataNotCrypt    bool
	LoginValidForever       bool

	ApplicationName string
	ApplicationDir  string
	ConfigDir       string
	TestFilesDir    string
	TmpDir          string
	MediaDir        string

	DB         Database
	MailConfig MailConfig

	WebURL WebLink

	Company Company
}
