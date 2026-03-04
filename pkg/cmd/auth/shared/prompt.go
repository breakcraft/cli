package shared

// Prompt defines an interface for interactive user prompts during authentication flows.
type Prompt interface {
	Select(string, string, []string) (int, error)
	Confirm(string, bool) (bool, error)
	InputHostname() (string, error)
	AuthToken() (string, error)
	Input(string, string) (string, error)
	Password(string) (string, error)
}
