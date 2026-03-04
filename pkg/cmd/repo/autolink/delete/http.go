package delete

import (
	"fmt"
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghinstance"
	"github.com/cli/cli/v2/internal/ghrepo"
)

// AutolinkDeleter is an HTTP client for deleting autolinks.
type AutolinkDeleter struct {
	HTTPClient *http.Client
}

// Delete removes an autolink from the given repository by ID.
func (a *AutolinkDeleter) Delete(repo ghrepo.Interface, id string) error {
	path := fmt.Sprintf("repos/%s/%s/autolinks/%s", repo.RepoOwner(), repo.RepoName(), id)
	url := ghinstance.RESTPrefix(repo.RepoHost()) + path
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("error deleting autolink: HTTP 404: Perhaps you are missing admin rights to the repository? (https://api.github.com/%s)", path)
	} else if resp.StatusCode > 299 {
		return api.HandleHTTPError(resp)
	}

	return nil
}
