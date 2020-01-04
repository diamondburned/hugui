package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/gen2brain/dlgs"
	"github.com/gohugoio/hugo/commands"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

func build() error {
	resp := commands.Execute([]string{"hugo"})
	return resp.Err
}

func test() error {
	var hugoErr = make(chan error)

	go func() {
		resp := commands.Execute(
			[]string{"serve", "-p", "34040", "-w"})
		hugoErr <- resp.Err
	}()

	select {
	case err := <-hugoErr:
		return errors.Wrap(err, "Hugo failed")
	case <-time.After(time.Second / 2):
		open.Run("http://127.0.0.1:34040/")
		return <-hugoErr
	}
}

func newPage(path string) error {
	d, err := ioutil.ReadDir(filepath.Join(path, "content"))
	if err != nil {
		return errors.Wrap(err, "Failed to read content/")
	}

	// Default location to create a new Hugo page
	var prefix = "posts/"

	if len(d) > 0 {
		const NewPage = "Create a new section..."

		var names = make([]string, len(d))
		for i, f := range d {
			if name := f.Name(); name != NewPage {
				names[i] = name
			}
		}

		// Allow choosing existing dirs
		c, ok, err := dlgs.List(DialogTitle, "Choose section",
			append(names, NewPage))
		must(err)

		if !ok {
			return ErrCancelled
		}

		// New location
		prefix = c
	}

	c, ok, err := dlgs.Entry(DialogTitle,
		"Enter a slug (dash-separated, characters only)", "example-slug")
	must(err)

	if !ok {
		return ErrCancelled
	}

	// Just to make sure
	c = strings.TrimSuffix(c, ".md")
	c = slugify(c)

	resp := commands.Execute([]string{"new", filepath.Join(prefix, c+".md")})
	if resp.Err != nil {
		return errors.Wrap(err, "Hugo error")
	}

	return nil
}

func slugify(name string) string {
	var dashed bool

	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}

		if !dashed {
			dashed = true
			return '-'
		}

		dashed = false
		return -1
	}, name)
}
