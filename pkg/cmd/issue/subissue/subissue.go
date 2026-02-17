package subissue

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghrepo"
	issueShared "github.com/cli/cli/v2/pkg/cmd/issue/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type SubIssueOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)

	ParentRef string
	ChildRef  string
	BeforeRef string
	AfterRef  string
}

func NewCmdSubIssue(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sub-issue <command>",
		Short: "Manage issue sub-issue relationships",
		Example: heredoc.Doc(`
			$ gh issue sub-issue add 10 20
			$ gh issue sub-issue remove 10 20
			$ gh issue sub-issue move 10 21 --before 20
		`),
	}

	cmd.AddCommand(
		newCmdAdd(f, nil),
		newCmdRemove(f, nil),
		newCmdMove(f, nil),
	)

	return cmd
}

func newCmdAdd(f *cmdutil.Factory, runF func(*SubIssueOptions) error) *cobra.Command {
	opts := &SubIssueOptions{IO: f.IOStreams, HttpClient: f.HttpClient}

	cmd := &cobra.Command{
		Use:   "add <parent> <child>",
		Short: "Add an issue as a sub-issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ParentRef = args[0]
			opts.ChildRef = args[1]
			opts.BaseRepo = f.BaseRepo
			if runF != nil {
				return runF(opts)
			}
			return runAdd(opts)
		},
	}

	return cmd
}

func newCmdRemove(f *cmdutil.Factory, runF func(*SubIssueOptions) error) *cobra.Command {
	opts := &SubIssueOptions{IO: f.IOStreams, HttpClient: f.HttpClient}

	cmd := &cobra.Command{
		Use:   "remove <parent> <child>",
		Short: "Remove a sub-issue relationship",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ParentRef = args[0]
			opts.ChildRef = args[1]
			opts.BaseRepo = f.BaseRepo
			if runF != nil {
				return runF(opts)
			}
			return runRemove(opts)
		},
	}

	return cmd
}

func newCmdMove(f *cmdutil.Factory, runF func(*SubIssueOptions) error) *cobra.Command {
	opts := &SubIssueOptions{IO: f.IOStreams, HttpClient: f.HttpClient}

	cmd := &cobra.Command{
		Use:   "move <parent> <child> (--before <sibling> | --after <sibling>)",
		Short: "Reorder a sub-issue under a parent issue",
		Args:  cobra.ExactArgs(2),
		Long: heredoc.Doc(`
			Reorder a sub-issue under a parent by specifying either --before or --after.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ParentRef = args[0]
			opts.ChildRef = args[1]
			opts.BaseRepo = f.BaseRepo

			if err := cmdutil.MutuallyExclusive(
				"specify only one of `--before` or `--after`",
				cmd.Flags().Changed("before"),
				cmd.Flags().Changed("after"),
			); err != nil {
				return err
			}
			if !cmd.Flags().Changed("before") && !cmd.Flags().Changed("after") {
				return cmdutil.FlagErrorf("specify one of `--before` or `--after`")
			}

			if runF != nil {
				return runF(opts)
			}
			return runMove(opts)
		},
	}

	cmd.Flags().StringVar(&opts.BeforeRef, "before", "", "Place the sub-issue before the specified sibling issue")
	cmd.Flags().StringVar(&opts.AfterRef, "after", "", "Place the sub-issue after the specified sibling issue")

	return cmd
}

func resolveIssue(client *api.Client, defaultRepo ghrepo.Interface, ref string) (*api.RESTIssueReference, ghrepo.Interface, error) {
	number, parsedRepoOpt, err := issueShared.ParseIssueFromArg(ref)
	if err != nil {
		return nil, nil, err
	}
	repo := defaultRepo
	if parsedRepo, ok := parsedRepoOpt.Value(); ok {
		repo = parsedRepo
	}
	issue, err := api.IssueByNumber(client, repo, number)
	if err != nil {
		return nil, nil, err
	}
	return issue, repo, nil
}

func runAdd(opts *SubIssueOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	apiClient := api.NewClientFromHTTP(httpClient)

	parentIssue, parentRepo, err := resolveIssue(apiClient, baseRepo, opts.ParentRef)
	if err != nil {
		return err
	}
	childIssue, _, err := resolveIssue(apiClient, parentRepo, opts.ChildRef)
	if err != nil {
		return err
	}

	_, err = api.AddIssueSubIssue(apiClient, parentRepo, parentIssue.Number, childIssue.ID, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.IO.Out, "✓ Added issue #%d as a sub-issue of #%d\n", childIssue.Number, parentIssue.Number)
	return nil
}

func runRemove(opts *SubIssueOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	apiClient := api.NewClientFromHTTP(httpClient)

	parentIssue, parentRepo, err := resolveIssue(apiClient, baseRepo, opts.ParentRef)
	if err != nil {
		return err
	}
	childIssue, _, err := resolveIssue(apiClient, parentRepo, opts.ChildRef)
	if err != nil {
		return err
	}

	_, err = api.RemoveIssueSubIssue(apiClient, parentRepo, parentIssue.Number, childIssue.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.IO.Out, "✓ Removed issue #%d from parent issue #%d\n", childIssue.Number, parentIssue.Number)
	return nil
}

func runMove(opts *SubIssueOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	apiClient := api.NewClientFromHTTP(httpClient)

	parentIssue, parentRepo, err := resolveIssue(apiClient, baseRepo, opts.ParentRef)
	if err != nil {
		return err
	}
	childIssue, _, err := resolveIssue(apiClient, parentRepo, opts.ChildRef)
	if err != nil {
		return err
	}

	var beforeID, afterID *int64
	if opts.BeforeRef != "" {
		beforeIssue, _, err := resolveIssue(apiClient, parentRepo, opts.BeforeRef)
		if err != nil {
			return err
		}
		beforeID = &beforeIssue.ID
	}
	if opts.AfterRef != "" {
		afterIssue, _, err := resolveIssue(apiClient, parentRepo, opts.AfterRef)
		if err != nil {
			return err
		}
		afterID = &afterIssue.ID
	}

	_, err = api.ReprioritizeIssueSubIssue(apiClient, parentRepo, parentIssue.Number, childIssue.ID, beforeID, afterID)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.IO.Out, "✓ Moved issue #%d under parent issue #%d\n", childIssue.Number, parentIssue.Number)
	return nil
}
