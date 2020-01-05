package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/dlgs"
	copier "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func pull(w *git.Worktree, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.PullContext(ctx,
		&git.PullOptions{Progress: os.Stderr}); err != nil {

		// Not really an error?
		if err == git.NoErrAlreadyUpToDate {
			dlgs.Info(DialogTitle, "Repository already up-to-date.")
			return nil
		}

		return errors.Wrap(err, "Failed to pull the tree")
	}

	return nil
}

func push(r *git.Repository, w *git.Worktree, path string) error {
	m, _, err := dlgs.Entry(DialogTitle,
		"Enter commit message", "Update")
	must(err)

	a, err := gitAuth(r)
	if err != nil {
		return err
	}

	_, err = w.Commit(m, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name: a.username,
			When: time.Now(),
		},
	})

	if err != nil {
		return errors.Wrap(err, "Failed to commit")
	}

	err = r.Push(&git.PushOptions{
		Progress: os.Stderr,
		Auth:     a.AuthMethod,
	})

	if err != nil {
		return errors.Wrap(err, "Failed to push to Git")
	}

	return nil
}

func clone(path string) error {
	url, ok, err := dlgs.Entry(DialogTitle, "Enter Git URL",
		"https://github.com/username/repository")
	must(err)

	if !ok {
		return errors.New("No repository entered")
	}

	// git-go will wipe our PWD because it's trash, so we're gonna pull a gamer
	// move.
	f, err := ioutil.TempDir(os.TempDir(), "hugui-")
	if err != nil {
		return errors.Wrap(err, "Failed to create temp dir")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// git clone, but we clone to that directory instead.
	_, err = git.PlainCloneContext(ctx, f, false,
		&git.CloneOptions{
			URL:      url,
			Progress: os.Stderr,
		},
	)

	if err != nil {
		return errors.Wrap(err, "Failed to clone")
	}

	// It worked, so now we should clean it up. After the function exits.
	defer os.RemoveAll(f)

	// Move the directory to where we want. We need to copy this first though.
	if err := copier.Copy(f, path); err != nil {
		return errors.Wrap(err, "Failed to copy cloned repository")
	}

	// Try adding the binary to gitignore
	if err := addGitignore(); err != nil {
		return errors.Wrap(err, "Failed to add gitignore")
	}

	// Stat for dialog
	s, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrap(err, "Stat failed")
	}

	_, err = dlgs.Info(DialogTitle, "Cloned successfully to "+s)
	return err
}

func addGitignore() error {
	h, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "Home not found")
	}

	h = filepath.Join(h, ".gitignore")

	f, err := os.OpenFile(h, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return errors.Wrap(err, "Failed to open ~/.gitignore")
	}

	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrap(err, "Failed to read")
	}

	// Already there? We're done.
	if strings.Contains(string(b), "hugui") {
		return nil
	}

	if _, err := fmt.Fprintf(f, "\nhugui\nhugui.exe\n"); err != nil {
		return errors.Wrap(err, "Failed to write")
	}

	return nil
}

type Auth struct {
	transport.AuthMethod
	username string
}

func gitAuth(r *git.Repository) (*Auth, error) {
	c, err := r.Config()
	if err != nil {
		return nil, errors.Wrap(err, "Can't get config")
	}

	origin, ok := c.Remotes["origin"]
	if !ok {
		return nil, errors.New("origin not found")
	}

	var auth transport.AuthMethod
	var user string

	if strings.HasPrefix(origin.URLs[0], "https://") {
		user, ok, err = dlgs.Entry(DialogTitle, "Enter Git username", "")
		if !ok {
			return nil, ErrCancelled
		}

		pass, ok, err := dlgs.Password(DialogTitle, "Enter Git password")
		must(err)

		if !ok {
			return nil, ErrCancelled
		}

		auth = &githttp.BasicAuth{
			Username: user, Password: pass,
		}
	} else {
		user, ok, err = dlgs.Entry(
			DialogTitle, "Enter Git username", "")
		must(err)

		if !ok {
			return nil, ErrCancelled
		}

		auth, err = gitssh.NewSSHAgentAuth(user)
		if err != nil {
			return nil, err
		}
	}

	return &Auth{
		AuthMethod: auth,
		username:   user,
	}, nil
}
