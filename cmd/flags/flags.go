package flagger

type CommandLineArgs struct {
	FrontendAPIHostAndPort string
	BackendAPIHostAndPort  string
	SqliteFilename         string
	Env                    string
}

var FLAGS *CommandLineArgs
