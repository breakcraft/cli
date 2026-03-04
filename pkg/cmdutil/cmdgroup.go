package cmdutil

import "github.com/spf13/cobra"

// AddGroup creates a new command group with the given title and adds the provided commands to it under the parent command.
func AddGroup(parent *cobra.Command, title string, cmds ...*cobra.Command) {
	g := &cobra.Group{
		Title: title,
		ID:    title,
	}
	parent.AddGroup(g)
	for _, c := range cmds {
		c.GroupID = g.ID
		parent.AddCommand(c)
	}
}
