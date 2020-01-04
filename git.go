package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/dlgs"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func pull(w *git.Worktree, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.PullContext(ctx,
		&git.PullOptions{Progress: os.Stderr}); err != nil {

		return errors.Wrap(err, "Failed to pull the tree")
	}

	return nil
}

func push(r *git.Repository, w *git.Worktree, path string) error {
	m, _, err := dlgs.Entry(DialogTitle,
		"Enter commit message", "Update")
	must(err)

	if err := gitPushInit(r); err != nil {
		return err
	}

	_, err = w.Commit(m, &git.CommitOptions{All: true})
	if err != nil {
		return errors.Wrap(err, "Failed to commit")
	}

	if err := r.Push(
		&git.PushOptions{Progress: os.Stderr}); err != nil {

		return errors.Wrap(err, "Failed to push to Git")
	}

	return nil
}

func clone(path string) error {
	path, ok, err := dlgs.Entry(DialogTitle, "Enter Git URL",
		"https://github.com/username/repository")
	must(err)

	if !ok {
		return errors.New("No repository entered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Cloning...")

	_, err = git.PlainCloneContext(ctx, path, false,
		&git.CloneOptions{
			URL:      path,
			Progress: os.Stderr,
		},
	)

	if err != nil {
		return errors.Wrap(err, "Failed to clone")
	}

	s, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrap(err, "Stat failed")
	}

	_, err = dlgs.Info(DialogTitle, "Cloned successfully to "+s)
	return err
}

func gitAuth(r *git.Repository) (transport.AuthMethod, error) {
	c, err := r.Config()
	if err != nil {
		return nil, errors.Wrap(err, "Can't get config")
	}

	origin, ok := c.Remotes["origin"]
	if !ok {
		return nil, errors.New("origin not found")
	}

	var auth transport.AuthMethod

	dn

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
		user, ok, err := dlgs.Entry(
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

	return auth, nil
}
