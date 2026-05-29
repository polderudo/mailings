package conf

type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Sender   string
	Insecure bool
	UseSSL   bool
	NoAuth   bool
}

type Database struct {
	User           string
	Password       string
	Host           string
	Port           int64
	Schema         string
	TimeoutSeconds int
}

type WebLink struct {
	WebURLLogin    string
	WebURL         string
	CreatePassword string
	EmailValidate  string
	APIUrl         string
}

type Company struct {
	Name string
}

type Logging struct {
	LogDir     string
	FileName   string
	FileSizeMB int
	MaxBackups int
	Compress   bool
	Disabled   bool
	NoColor    bool

	AlwaysShowDebugInConsole bool

	LogDebug bool

	ErrorLogging *Logging
	WebLogging   *Logging
	DebugLogging *Logging
}
