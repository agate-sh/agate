package version

var (
	// Version is set via ldflags during build
	Version = "dev"
)

// Short returns the version string
func Short() string {
	return Version
}