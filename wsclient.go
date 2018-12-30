package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"

	"github.com/galdor/go-cmdline"
)

func main() {
	// Command line
	cmdline := cmdline.New()
	cmdline.AddArgument("uri", "the uri of the websocket endpoint")
	cmdline.Parse(os.Args)

	uri, err := url.Parse(cmdline.ArgumentValue("uri"))
	if err != nil {
		die("invalid uri: %v", err)
	}

	// Client
	client := NewClient(uri)

	if err := client.Start(); err != nil {
		die("cannot connect to %s: %v", uri.String(), err)
	}
	defer client.Stop()

	// Input
	go func() {
		inputReader := bufio.NewReader(os.Stdin)

		for {
			printPrompt()

			line, err := inputReader.ReadString('\n')
			if err != nil {
				die("cannot read standard input: %v", err)
			}

			client.SendChan <- line
		}
	}()

	// Main
loop:
	for {
		select {
		case msg := <-client.RecvChan:
			clearPrompt()
			fmt.Printf("%s\n", string(msg.Data))
			printPrompt()

		case err := <-client.ErrorChan:
			clearPrompt()
			fmt.Fprintf(os.Stderr, "error: %v\n> ", err)
			break loop
		}
	}
}

func clearPrompt() {
	fmt.Printf("\033[G\033[K")
}

func printPrompt() {
	fmt.Printf("> ")
}

func info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func die(format string, args ...interface{}) {
	warn(format, args...)
	os.Exit(1)
}
