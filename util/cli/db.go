package cli

// DBOptions contains common flags for commands using the DB
type DBOptions struct {
	DB string `long:"db" default:"postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable" env:"LOOKOUT_DB" description:"connection string to postgres database"`
}
