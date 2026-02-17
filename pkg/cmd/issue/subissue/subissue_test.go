package subissue

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func runSubIssueCommand(t *testing.T, rt http.RoundTripper, cli string) (string, error) {
	t.Helper()
	ios, _, stdout, _ := iostreams.Test()
	ios.SetStdoutTTY(false)
	ios.SetStdinTTY(false)
	ios.SetStderrTTY(false)

	factory := &cmdutil.Factory{
		IOStreams: ios,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: rt}, nil
		},
		Config: func() (gh.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (ghrepo.Interface, error) {
			return ghrepo.New("OWNER", "REPO"), nil
		},
	}

	cmd := NewCmdSubIssue(factory)
	argv, err := shlex.Split(cli)
	if err != nil {
		return "", err
	}
	cmd.SetArgs(argv)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err = cmd.ExecuteC()
	return stdout.String(), err
}

func TestSubIssueAdd(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/10"), httpmock.StringResponse(`{"id":1,"number":10,"html_url":"https://github.com/OWNER/REPO/issues/10","state":"open"}`))
	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/20"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))
	reg.Register(httpmock.REST("POST", "repos/OWNER/REPO/issues/10/sub_issues"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))

	out, err := runSubIssueCommand(t, reg, "add 10 20")
	assert.NoError(t, err)
	assert.Contains(t, out, "Added issue #20 as a sub-issue of #10")
}

func TestSubIssueRemove(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/10"), httpmock.StringResponse(`{"id":1,"number":10,"html_url":"https://github.com/OWNER/REPO/issues/10","state":"open"}`))
	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/20"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))
	reg.Register(httpmock.REST("DELETE", "repos/OWNER/REPO/issues/10/sub_issue"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))

	out, err := runSubIssueCommand(t, reg, "remove 10 20")
	assert.NoError(t, err)
	assert.Contains(t, out, "Removed issue #20 from parent issue #10")
}

func TestSubIssueMove(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/10"), httpmock.StringResponse(`{"id":1,"number":10,"html_url":"https://github.com/OWNER/REPO/issues/10","state":"open"}`))
	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/20"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))
	reg.Register(httpmock.REST("GET", "repos/OWNER/REPO/issues/21"), httpmock.StringResponse(`{"id":3,"number":21,"html_url":"https://github.com/OWNER/REPO/issues/21","state":"open"}`))
	reg.Register(httpmock.REST("PATCH", "repos/OWNER/REPO/issues/10/sub_issues/priority"), httpmock.StringResponse(`{"id":2,"number":20,"html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`))

	out, err := runSubIssueCommand(t, reg, "move 10 20 --before 21")
	assert.NoError(t, err)
	assert.Contains(t, out, "Moved issue #20 under parent issue #10")
}

func TestSubIssueMoveValidation(t *testing.T) {
	reg := &httpmock.Registry{}
	out, err := runSubIssueCommand(t, reg, "move 10 20")
	assert.Error(t, err)
	assert.Equal(t, "", out)
}
