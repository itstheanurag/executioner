package languages

type RuntimeConfig struct {
	Image          string
	SourceFile     string
	CompileCommand []string
	RunCommand     []string
}

type Language struct {
	ID     string
	Name   string
	Config RuntimeConfig
}
