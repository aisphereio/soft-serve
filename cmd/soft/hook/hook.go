package hook

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/aisphereio/soft-serve/cmd"
	"github.com/aisphereio/soft-serve/pkg/backend"
	"github.com/aisphereio/soft-serve/pkg/config"
	"github.com/aisphereio/soft-serve/pkg/hooks"
	"github.com/spf13/cobra"
)

var (
	// ErrInternalServerError indicates that an internal server error occurred.
	ErrInternalServerError = errors.New("internal server error")

	// Deprecated: this flag is ignored.
	configPath string

	// Command is the hook command.
	Command = &cobra.Command{
		Use:    "hook",
		Short:  "Run git server hooks",
		Long:   "Handles Soft Serve git server hooks.",
		Hidden: true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			logger := log.FromContext(c.Context())
			if err := cmd.InitBackendContext(c, args); err != nil {
				logger.Error("failed to initialize backend context", "err", err)
				return ErrInternalServerError
			}

			return nil
		},
		PersistentPostRunE: func(c *cobra.Command, args []string) error {
			logger := log.FromContext(c.Context())
			if err := cmd.CloseDBContext(c, args); err != nil {
				logger.Error("failed to close backend", "err", err)
				return ErrInternalServerError
			}

			return nil
		},
	}

	// Git hooks read the config from the environment, based on
	// $SOFT_SERVE_DATA_PATH. We already parse the config when the binary
	// starts, so we don't need to do it again.
	// The --config flag is now deprecated.
	hooksRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		hks := backend.FromContext(ctx)
		cfg := config.FromContext(ctx)

		// This is set in the server before invoking git-receive-pack/git-upload-pack
		repoName := os.Getenv("SOFT_SERVE_REPO_NAME")

		logger := log.FromContext(ctx).With("repo", repoName)

		stdin := cmd.InOrStdin()
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()

		cmdName := cmd.Name()
		customHookPath := filepath.Join(cfg.DataPath, "hooks", cmdName)

		input, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("read %s hook input: %w", cmdName, err)
		}
		if err := runInternalHook(ctx, hks, cmdName, repoName, bytes.NewReader(input), stdout, stderr, args); err != nil {
			logger.Error("internal hook rejected update", "hook", cmdName, "err", err)
			return err
		}

		// Custom hooks
		if stat, err := os.Stat(customHookPath); err == nil && !stat.IsDir() && stat.Mode()&0o111 != 0 {
			// If the custom hook is executable, run it
			if err := runCommand(ctx, bytes.NewReader(input), stdout, stderr, customHookPath, args...); err != nil {
				logger.Error("failed to run custom hook", "err", err)
			}
		}

		return nil
	}

	preReceiveCmd = &cobra.Command{
		Use:   "pre-receive",
		Short: "Run git pre-receive hook",
		RunE:  hooksRunE,
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Run git update hook",
		Args:  cobra.ExactArgs(3),
		RunE:  hooksRunE,
	}

	postReceiveCmd = &cobra.Command{
		Use:   "post-receive",
		Short: "Run git post-receive hook",
		RunE:  hooksRunE,
	}

	postUpdateCmd = &cobra.Command{
		Use:   "post-update",
		Short: "Run git post-update hook",
		RunE:  hooksRunE,
	}
)

func runInternalHook(ctx context.Context, hks hooks.Hooks, cmdName, repoName string, stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	switch cmdName {
	case hooks.PreReceiveHook, hooks.PostReceiveHook:
		opts := make([]hooks.HookArg, 0)
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) != 3 {
				return fmt.Errorf("invalid %s hook input %q", cmdName, scanner.Text())
			}
			opts = append(opts, hooks.HookArg{
				OldSha:  fields[0],
				NewSha:  fields[1],
				RefName: fields[2],
			})
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read %s hook input: %w", cmdName, err)
		}
		if cmdName == hooks.PreReceiveHook {
			return hks.PreReceive(ctx, stdout, stderr, repoName, opts)
		}
		hks.PostReceive(ctx, stdout, stderr, repoName, opts)
		return nil
	case hooks.UpdateHook:
		if len(args) != 3 {
			return fmt.Errorf("invalid update hook input: got %d arguments", len(args))
		}
		return hks.Update(ctx, stdout, stderr, repoName, hooks.HookArg{
			RefName: args[0],
			OldSha:  args[1],
			NewSha:  args[2],
		})
	case hooks.PostUpdateHook:
		hks.PostUpdate(ctx, stdout, stderr, repoName, args...)
		return nil
	default:
		return fmt.Errorf("unsupported hook %q", cmdName)
	}
}

func init() {
	Command.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (deprecated)")
	Command.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
		postUpdateCmd,
	)
}

func runCommand(ctx context.Context, in io.Reader, out io.Writer, err io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err
	return cmd.Run()
}
