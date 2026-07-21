package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/aisphereio/soft-serve/cmd/soft/admin"
	"github.com/aisphereio/soft-serve/cmd/soft/browse"
	"github.com/aisphereio/soft-serve/cmd/soft/hook"
	"github.com/aisphereio/soft-serve/cmd/soft/serve"
	"github.com/aisphereio/soft-serve/pkg/config"
	logr "github.com/aisphereio/soft-serve/pkg/log"
	"github.com/aisphereio/soft-serve/pkg/ui/common"
	"github.com/aisphereio/soft-serve/pkg/version"
	"github.com/muesli/mango-cobra"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
	_ "modernc.org/sqlite"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags
	// when building.
	CommitSHA = ""

	// CommitDate contains the date of the commit that this application was built
	// against. It's set via ldflags
	// when building.
	CommitDate = ""

	rootCmd = &cobra.Command{
		Use:          "soft",
		Short:        "A self-hostable Git server for the command line",
		Long:         "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return browse.Command.RunE(cmd, args)
		},
	}

	manCmd = &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			manPage, err := mcobra.NewManPage(1, rootCmd) //.
			if err != nil {
				return err
			}

			manPage = manPage.WithSection("Copyright", "(C) 2021-2023 Charmbracelet, Inc.\n"+
				"Released under MIT license.")
			fmt.Println(manPage.Build(roff.NewDocument()))
			return nil
		},
	}
)

func init() {
	if noColor, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_NO_COLOR")); noColor {
		common.DefaultColorProfile = colorprofile.NoTTY
	}

	rootCmd.AddCommand(
		manCmd,
		serve.Command,
		hook.Command,
		admin.Command,
		browse.Command,
	)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
