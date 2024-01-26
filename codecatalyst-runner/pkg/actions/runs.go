package actions

// Runs defines the runtime environment and main entry point for the action.
type Runs struct {
	Using UsingType `yaml:"Using"` // specify the type of runtime environment. Currently, Node 12, Node 16 and Docker are the options

	// Using == 'node16'
	Main string `yaml:"Main"` // specify the file for the entry point of a Node.js application. This file contains your action code. Required if Node 12 or Node 16 runtime is specified for Using
	Pre  string `yaml:"Pre"`  // allows you to run a script at the beginning of the action run. Can be defined if Node 12 or Node 16 runtime is specified for Using
	Post string `yaml:"Post"` // allows your to run a script at the end of the action run. Can be defined if Node 12 or Node 16 runtime is specified for Using

	// Using == 'docker'
	Image          string `yaml:"Image"`          // specify the file or link to an image. If a link is specified, it's not validated. This is the Docker image used as the container to run the action. Required if Docker runtime is specified for Using
	Entrypoint     string `yaml:"Entrypoint"`     // overrides the Docker entrypoint in the Docker file. Can be defined if Docker runtime is specified for Using.
	PreEntryPoint  string `yaml:"PreEntryPoint"`  // allows you to run a script before the entrypoint action begins. Can be defined if Docker runtime is specified for Using.
	PostEntryPoint string `yaml:"PostEntryPoint"` // allows you to run a cleanup script once the entrypoint action has finished. Can be defined if Docker runtime is specified for Using.
}

// UsingType specifies the type of runtime environment. Currently, Node 12, Node 16 and Docker are the options.
type UsingType string

const (
	// UsingTypeNode12 specifies the Node 12 runtime environment.
	UsingTypeNode12 UsingType = "node12"
	// UsingTypeNode16 specifies the Node 16 runtime environment.
	UsingTypeNode16 UsingType = "node16"
	// UsingTypeDocker specifies the Docker runtime environment.
	UsingTypeDocker UsingType = "docker"
)
