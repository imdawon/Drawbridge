package flagger

type CommandLineArgs struct {
	DrawbridgePort         uint // The actual Drawbridge server port that Emissary clients will connect to.
	FrontendAPIHostAndPort string
	BackendAPIHostAndPort  string
	SqliteFilename         string
	Env                    string
}

var FLAGS *CommandLineArgs
