package version

// Version is the build version, overridable via:
//
//	go build -ldflags "-X github.com/shhac/git-wt/internal/version.Version=X.Y.Z"
var Version = "0.7.0-dev"
