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

// Postmark bündelt die Konfiguration für den Newsletter-Versand über die
// Postmark Bulk-Email-API (POST /email/bulk + GET /email/bulk/{id}).
type Postmark struct {
	ServerToken string // X-Postmark-Server-Token
	ApiBaseURL  string // Default https://api.postmarkapp.com (wird in NewClient gesetzt, falls leer)
	TrackOpens  bool
	TrackLinks  string // "None" | "HtmlAndText" | "HtmlOnly" | "TextOnly"
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
