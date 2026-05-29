package auth

const (
	ActionUpsert = "upsert"
)

const (
	OpAll       = "*"
	OpAllSuper  = "**"
	OpConfig    = "config"
	OpDeveloper = "developer"
	OpMailing   = "mailing"
)

func GetAllOperations() []string {
	return []string{
		OpConfig,
		OpDeveloper,
		OpMailing,
	}
}
