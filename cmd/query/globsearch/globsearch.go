package globsearch

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	"github.com/bitbomdev/minefield/cmd/helpers"
	apiv1 "github.com/bitbomdev/minefield/gen/api/v1"
	"github.com/bitbomdev/minefield/gen/api/v1/apiv1connect"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type options struct {
	maxOutput          int
	addr               string
	output             string
	graphServiceClient apiv1connect.GraphServiceClient
}

// AddFlags adds command-line flags to the provided cobra command.
func (o *options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&o.maxOutput, "max-output", 10, "maximum number of results to display")
	cmd.Flags().StringVar(&o.addr, "addr", "http://localhost:8089", "address of the minefield server")
	cmd.Flags().StringVar(&o.output, "output", "table", "output format (table or json)")
}

// Run executes the globsearch command with the provided arguments.
func (o *options) Run(cmd *cobra.Command, args []string) error {
	pattern := args[0]
	if pattern == "" {
		return fmt.Errorf("pattern is required")
	}

	// Initialize client if not injected (for testing)
	if o.graphServiceClient == nil {
		o.graphServiceClient = apiv1connect.NewGraphServiceClient(
			http.DefaultClient,
			o.addr,
		)
	}

	// Query nodes matching pattern
	res, err := o.graphServiceClient.GetNodesByGlob(
		cmd.Context(),
		connect.NewRequest(&apiv1.GetNodesByGlobRequest{Pattern: pattern}),
	)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	if len(res.Msg.Nodes) == 0 {
		return fmt.Errorf("no nodes found matching pattern: %s", pattern)
	}

	// Format and display results
	switch o.output {
	case "json":
		jsonOutput, err := helpers.FormatNodeJSON(res.Msg.Nodes)
		if err != nil {
			return fmt.Errorf("failed to format nodes as JSON: %w", err)
		}
		cmd.Println(string(jsonOutput))
		return nil
	case "table":
		return formatTable(cmd.OutOrStdout(), res.Msg.Nodes, o.maxOutput)
	default:
		return fmt.Errorf("unknown output format: %s", o.output)
	}
}

// formatTable formats the nodes into a table and writes it to the provided writer.
func formatTable(w io.Writer, nodes []*apiv1.Node, maxOutput int) error {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Name", "Type", "ID"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)

	for i, node := range nodes {
		if i >= maxOutput {
			break
		}
		table.Append([]string{
			node.Name,
			node.Type,
			strconv.FormatUint(uint64(node.Id), 10),
		})
	}

	table.Render()
	return nil
}

// New returns a new cobra command for globsearch.
func New() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:               "globsearch [pattern]",
		Short:             "Search for nodes by glob pattern",
		Long:              "Search for nodes in the graph using a glob pattern",
		Args:              cobra.ExactArgs(1),
		RunE:              o.Run,
		DisableAutoGenTag: true,
	}
	o.AddFlags(cmd)
	return cmd
}
