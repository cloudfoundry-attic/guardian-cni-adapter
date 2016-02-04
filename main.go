package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/guardian-cni-adapter/controller"
)

var (
	action           string
	handle           string
	networkSpec      string
	cniPluginDir     string
	cniConfigDir     string
	ducatiSandboxDir string
	daemonBaseURL    string
)

func parseArgs(allArgs []string) error {
	action = allArgs[1]

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&networkSpec, "network", "", "")
	flagSet.StringVar(&cniPluginDir, "cniPluginDir", "", "")
	flagSet.StringVar(&cniConfigDir, "cniConfigDir", "", "")
	flagSet.StringVar(&ducatiSandboxDir, "ducatiSandboxDir", "", "")
	flagSet.StringVar(&daemonBaseURL, "daemonBaseURL", "", "")

	err := flagSet.Parse(allArgs[2:])
	if err != nil {
		os.Exit(1) // exit, error was already printed to stderr by flagSet.Parse
	}
	if len(flagSet.Args()) > 0 {
		log.Fatalf("unexpected extra args: %+v", flagSet.Args())
	}

	if handle == "" {
		log.Fatalf("missing required flag 'handle'")
	}
	if cniPluginDir == "" {
		log.Fatalf("missing required flag 'cniPluginDir'")
	}
	if cniConfigDir == "" {
		log.Fatalf("missing required flag 'cniConfigDir'")
	}
	if ducatiSandboxDir == "" {
		log.Fatalf("missing required flag 'ducatiSandboxDir'")
	}
	if daemonBaseURL == "" {
		log.Fatalf("missing required flag 'daemonBaseURL'")
	}
	if action == "up" && networkSpec == "" {
		log.Fatalf("missing required flag 'network'")
	}

	return nil
}

func main() {
	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		log.Fatalf("this is a OCI prestart/poststop hook.  see https://github.com/opencontainers/specs/blob/master/runtime-config.md")
	}

	inputBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("unable to read stdin: %s", err)
	}

	var containerState struct {
		Pid int
	}
	err = json.Unmarshal(inputBytes, &containerState)
	if err != nil {
		log.Fatalf("input is not valid json: %s: %q", err, string(inputBytes))
	}
	if containerState.Pid == 0 {
		log.Fatalf("missing pid")
	}

	err = parseArgs(os.Args)
	if err != nil {
		log.Fatalf("arg parsing error: %s", err)
	}

	myController := controller.Controller{
		PluginDir:      cniPluginDir,
		ConfigDir:      cniConfigDir,
		SandboxDirPath: ducatiSandboxDir,
		DaemonBaseURL:  daemonBaseURL,
	}

	namespacePath := fmt.Sprintf("/proc/%d/ns/net", containerState.Pid)

	switch action {
	case "up":
		err = myController.Up(namespacePath, handle, "some-spec-not-set")
		if err != nil {
			log.Fatalf("up failed: %s", err)
		}
	case "down":
		err = myController.Down(namespacePath, handle)
		if err != nil {
			log.Fatalf("down failed: %s", err)
		}
	default:
		log.Fatalf("action: %s is unrecognized", action)
	}
}
