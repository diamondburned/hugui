package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gen2brain/dlgs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cli "github.com/urfave/cli/v2"
	"gopkg.in/src-d/go-git.v4"
)

const DialogTitle = "hugui - Hugo GUI"

var ErrCancelled = errors.New("Cancelled")

func main() {
	app := &cli.App{
		Name:   "hugui",
		Action: Main,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Value: ".",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		if err == ErrCancelled {
			dlgs.Info(DialogTitle, "Cancelled.")
			return
		}

		fatal(err)
	}
}

func Main(c *cli.Context) error {
	var path = c.String("path")

	choice, ok, err := dlgs.List(DialogTitle, "Choose action", []string{
		"Git: Clone (initialize)",
		"Git: Update/Pull",
		"Git: Push",
		"Hugo: Build the website to public/",
		"Hugo: Test the website",
		"Hugo: New page",
	})
	must(err)

	if !ok {
		return ErrCancelled
	}

	if strings.HasPrefix(choice, "Git: ") {
		if choice == "Git: Clone (initialize)" {
			return clone(path)
		}

		r, err := git.PlainOpen(path)
		if err != nil {
			return errors.Wrap(err, "Failed to open "+path)
		}

		w, err := r.Worktree()
		if err != nil {
			return errors.Wrap(err, "Failed to get worktree")
		}

		switch choice {
		case "Git: Update/Pull":
			return pull(w, path)
		case "Git: Push":
			return push(r, w, path)
		default:
			fatal("Unknown Git command (BUG)")
		}
	}

	// cd into the path, so hugo works properly
	if err := os.Chdir(path); err != nil {
		return errors.Wrap(err, "Failed to chdir")
	}

	// We're fine with Hugo being ran in cmd.exe
	cobra.MousetrapHelpText = ""

	switch choice {
	case "Hugo: Build the website to public/":
		return build()
	case "Hugo: Test the website":
		return test()
	case "Hugo: New page":
		return newPage(path)

	default:
		fatal("Unknown (BUG)")
	}

	return nil
}

func must(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func fatal(v ...interface{}) {
	s := fmt.Sprintln(v...)
	dlgs.Error(DialogTitle+": Error", s)
	log.Panicln(s)
}
