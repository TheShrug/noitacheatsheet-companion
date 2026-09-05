// Command companion is the noitacheatsheet.com desktop uploader.
//
// Two commands today. "paths" finds Noita's data folder on this machine and
// reports what it found — the foundation everything else stands on, since a
// watcher pointed at the wrong folder fails silently, and the check to run
// first when the app is not seeing your clips. "serve" opens the review queue
// on 127.0.0.1.
//
// The watcher is not wired into "serve" yet, and nothing uploads. See the
// repository's open issues.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/TheShrug/noitacheatsheet-companion/internal/noita"
	"github.com/TheShrug/noitacheatsheet-companion/internal/queue"
	"github.com/TheShrug/noitacheatsheet-companion/internal/server"
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
	port := fs.Int("port", server.DefaultPort, "port for the review queue, which is served on 127.0.0.1 and never anywhere else")
	showVersion := fs.Bool("version", false, "print the version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "companion — the noitacheatsheet.com desktop uploader\n\n")
		fmt.Fprintf(fs.Output(), "Usage:\n")
		fmt.Fprintf(fs.Output(), "  companion paths [--root DIR]   report where Noita's files are on this machine\n")
		fmt.Fprintf(fs.Output(), "  companion serve [--port N]     serve the review queue on 127.0.0.1\n\n")
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
	case "serve":
		return serveQueue(*port)
	case "":
		fs.Usage()
		return errors.New("\nno command given")
	default:
		fs.Usage()
		return fmt.Errorf("\nunknown command %q", command)
	}
}

// serveQueue serves the review queue until the process is stopped.
//
// It reads the queue file and nothing else: the watcher that fills that file
// is not wired in here yet (issue #11), so this shows whatever has been queued
// and offers to confirm it. Confirming is local — there is nowhere to upload
// to yet (ADR 2).
func serveQueue(port int) error {
	path, err := queue.ConfigPath()
	if err != nil {
		return err
	}

	q := queue.Load(path)
	srv, err := server.New(q, path, port)
	if err != nil {
		return err
	}

	// Bind before printing anything, so a port already in use is an error
	// rather than a URL that does not answer.
	ln, err := srv.Listen()
	if err != nil {
		return err
	}

	fmt.Printf("Queue file  %s\n", path)
	fmt.Printf("Clips       %d waiting for review\n", len(q.Entries()))
	fmt.Println("The clip watcher is not wired into this command yet (issue #11), so")
	fmt.Println("this serves what is already in the queue file.")
	fmt.Println()

	// The URL is the last line on startup: the fleet's contract, and what the
	// tray's "Open queue" will open (ADR 3).
	fmt.Println(srv.URL())

	return srv.Serve(ln)
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
