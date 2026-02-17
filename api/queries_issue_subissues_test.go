package api

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueParent(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("GET", "repos/OWNER/REPO/issues/123/parent"),
		httpmock.StringResponse(`{"id":1,"number":10,"title":"parent","html_url":"https://github.com/OWNER/REPO/issues/10","state":"open"}`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	issue, err := IssueParent(client, repo, 123)
	require.NoError(t, err)
	assert.Equal(t, 10, issue.Number)
	assert.Equal(t, "https://github.com/OWNER/REPO/issues/10", issue.URL)
}

func TestIssueSubIssues(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("GET", "repos/OWNER/REPO/issues/10/sub_issues"),
		httpmock.StringResponse(`[{"id":2,"number":20,"title":"child","html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}]`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	issues, err := IssueSubIssues(client, repo, 10)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, 20, issues[0].Number)
}

func TestIssueByNumber(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("GET", "repos/OWNER/REPO/issues/20"),
		httpmock.StringResponse(`{"id":2,"number":20,"title":"child","html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	issue, err := IssueByNumber(client, repo, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), issue.ID)
	assert.Equal(t, 20, issue.Number)
}

func TestAddIssueSubIssue(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("POST", "repos/OWNER/REPO/issues/10/sub_issues"),
		httpmock.StringResponse(`{"id":2,"number":20,"title":"child","html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	issue, err := AddIssueSubIssue(client, repo, 10, 2, true)
	require.NoError(t, err)
	assert.Equal(t, 20, issue.Number)

	reqBody, err := io.ReadAll(reg.Requests[0].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub_issue_id":2,"replace_parent":true}`, string(reqBody))
}

func TestRemoveIssueSubIssue(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("DELETE", "repos/OWNER/REPO/issues/10/sub_issue"),
		httpmock.StringResponse(`{"id":2,"number":20,"title":"child","html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	issue, err := RemoveIssueSubIssue(client, repo, 10, 2)
	require.NoError(t, err)
	assert.Equal(t, 20, issue.Number)

	reqBody, err := io.ReadAll(reg.Requests[0].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub_issue_id":2}`, string(reqBody))
}

func TestReprioritizeIssueSubIssue(t *testing.T) {
	reg := &httpmock.Registry{}
	defer reg.Verify(t)

	reg.Register(
		httpmock.REST("PATCH", "repos/OWNER/REPO/issues/10/sub_issues/priority"),
		httpmock.StringResponse(`{"id":2,"number":20,"title":"child","html_url":"https://github.com/OWNER/REPO/issues/20","state":"open"}`),
	)

	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	before := int64(3)
	issue, err := ReprioritizeIssueSubIssue(client, repo, 10, 2, &before, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, issue.Number)

	reqBody, err := io.ReadAll(reg.Requests[0].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub_issue_id":2,"before_id":3}`, string(reqBody))
}

func TestReprioritizeIssueSubIssue_Validation(t *testing.T) {
	reg := &httpmock.Registry{}
	client := newTestClient(reg)
	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	before := int64(3)
	after := int64(4)

	_, err := ReprioritizeIssueSubIssue(client, repo, 10, 2, nil, nil)
	require.EqualError(t, err, "exactly one of beforeID or afterID must be provided")

	_, err = ReprioritizeIssueSubIssue(client, repo, 10, 2, &before, &after)
	require.EqualError(t, err, "exactly one of beforeID or afterID must be provided")
}

func TestSubIssuesMarshalJSON(t *testing.T) {
	t.Run("nil nodes marshals as empty array", func(t *testing.T) {
		b, err := json.Marshal(SubIssues{})
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(b))
	})

	t.Run("nodes marshals as node array", func(t *testing.T) {
		b, err := json.Marshal(SubIssues{
			Nodes: []IssueTreeNode{{Number: 42}},
		})
		require.NoError(t, err)
		var decoded []map[string]interface{}
		err = json.Unmarshal(b, &decoded)
		require.NoError(t, err)
		require.Len(t, decoded, 1)
		assert.Equal(t, float64(42), decoded[0]["number"])
	})
}

func TestSubIssuesUnmarshalJSON(t *testing.T) {
	t.Run("object payload", func(t *testing.T) {
		var s SubIssues
		err := json.Unmarshal([]byte(`{"nodes":[{"number":7}],"totalCount":9}`), &s)
		require.NoError(t, err)
		assert.Equal(t, 1, len(s.Nodes))
		assert.Equal(t, 9, s.TotalCount)
	})

	t.Run("array payload", func(t *testing.T) {
		var s SubIssues
		err := json.Unmarshal([]byte(`[{"number":7}]`), &s)
		require.NoError(t, err)
		assert.Equal(t, 1, len(s.Nodes))
		assert.Equal(t, 1, s.TotalCount)
	})
}
