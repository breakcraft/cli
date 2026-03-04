package shared

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cli/cli/v2/internal/text"
)

// Visibility represents the visibility scope of a secret.
type Visibility string

const (
	// All indicates the secret is visible to all repositories.
	All = "all"
	// Private indicates the secret is visible to private repositories.
	Private = "private"
	// Selected indicates the secret is visible to selected repositories.
	Selected = "selected"
)

// App represents the application type for a secret.
type App string

const (
	// Actions represents the GitHub Actions application.
	Actions = "actions"
	// Codespaces represents the GitHub Codespaces application.
	Codespaces = "codespaces"
	// Dependabot represents the GitHub Dependabot application.
	Dependabot = "dependabot"
	// Unknown represents an unrecognized application.
	Unknown = "unknown"
)

// String returns the string representation of the App.
func (app App) String() string {
	return string(app)
}

// Title returns the title-cased name of the App.
func (app App) Title() string {
	return text.Title(app.String())
}

// SecretEntity represents the level at which a secret is stored.
type SecretEntity string

const (
	// Repository indicates a repository-level secret.
	Repository = "repository"
	// Organization indicates an organization-level secret.
	Organization = "organization"
	// User indicates a user-level secret.
	User = "user"
	// Environment indicates an environment-level secret.
	Environment = "environment"
)

// GetSecretEntity determines the secret entity from the provided flags.
func GetSecretEntity(orgName, envName string, userSecrets bool) (SecretEntity, error) {
	orgSet := orgName != ""
	envSet := envName != ""

	if orgSet && envSet || orgSet && userSecrets || envSet && userSecrets {
		return "", errors.New("cannot specify multiple secret entities")
	}

	if orgSet {
		return Organization, nil
	}
	if envSet {
		return Environment, nil
	}
	if userSecrets {
		return User, nil
	}
	return Repository, nil
}

// GetSecretApp determines the application type for a secret.
func GetSecretApp(app string, entity SecretEntity) (App, error) {
	switch strings.ToLower(app) {
	case Actions:
		return Actions, nil
	case Codespaces:
		return Codespaces, nil
	case Dependabot:
		return Dependabot, nil
	case "":
		if entity == User {
			return Codespaces, nil
		}
		return Actions, nil
	default:
		return Unknown, fmt.Errorf("invalid application: %s", app)
	}
}

// IsSupportedSecretEntity reports whether the app supports the given entity.
func IsSupportedSecretEntity(app App, entity SecretEntity) bool {
	switch app {
	case Actions:
		return entity == Repository || entity == Organization || entity == Environment
	case Codespaces:
		return entity == User || entity == Organization || entity == Repository
	case Dependabot:
		return entity == Repository || entity == Organization
	default:
		return false
	}
}
