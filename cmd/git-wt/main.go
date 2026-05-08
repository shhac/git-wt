package main

import "github.com/shhac/git-wt/internal/cli"

// version is overridden via ldflags at release time:
//
//	go build -ldflags "-X main.version=X.Y.Z" ./cmd/git-wt
//
// Unbuilt invocations (`go run`) keep the literal "dev".
var version = "dev"

func main() {
	cli.Execute(version)
}
