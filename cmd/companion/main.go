// Command companion is the noitacheatsheet.com desktop uploader.
//
// Today it does one thing: find Noita's data folder on this machine and report
// what it found. That is the foundation everything else stands on — a watcher
// pointed at the wrong folder fails silently — and it is the check to run first
// when the app is not seeing your clips.
//
// The watcher, the review queue and the upload are not built yet. See the
// repository's open issues.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/TheShrug/noitacheatsheet-companion/internal/noita"
)

// version is stamped by the linker at release time; see the Makefile.
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("companion", flag.ContinueOnError)
	root := fs.String("root", "", "Noita data folder to use instead of searching (the one containing save00 and save_rec)")
	showVersion := fs.Bool("version", false, "print the version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "companion — the noitacheatsheet.com desktop uploader\n\n")
		fmt.Fprintf(fs.Output(), "Usage:\n  companion paths [--root DIR]   report where Noita's files are on this machine\n\n")
		fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
	}

	// Flags first so --version and --help work without a subcommand.
	command := ""
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		command, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		fmt.Println("companion", version)
		return nil
	}

	switch command {
	case "paths":
		return printPaths(*root)
	case "":
		fs.Usage()
		return errors.New("\nno command given")
	default:
		fs.Usage()
		return fmt.Errorf("\nunknown command %q", command)
	}
}

// printPaths reports the resolved install, or every place that was searched.
//
// This is deliberately verbose on failure. The candidate list for Linux is
// assembled from public documentation rather than from a Proton install anyone
// has run, so a player pasting this output is how that list gets corrected.
func printPaths(root string) error {
	var (
		install noita.Install
		err     error
	)

	if root != "" {
		install, err = noita.At(root)
	} else {
		install, err = noita.Detect()
	}

	if err != nil {
		if errors.Is(err, noita.ErrNotFound) {
			return errors.New(noita.ExplainNotFound(noita.ProbeAll()))
		}
		return err
	}

	fmt.Printf("Platform    %s\n", install.Platform)
	fmt.Printf("Data root   %s\n", install.Root)
	fmt.Printf("Active save %s\n", install.Save)
	fmt.Printf("player.xml  %s\n", install.PlayerXML())
	fmt.Printf("Clips       %s\n", install.Gifs)

	// The clip folder only appears once the in-game recorder has written to
	// it, so a missing one is a normal state and worth naming rather than
	// leaving as a path that happens not to exist.
	if _, statErr := os.Stat(install.Gifs); statErr != nil {
		fmt.Println("\nThe clip folder does not exist yet. Noita creates it the first time")
		fmt.Println("the in-game recorder saves a gif.")
	}

	return nil
}
