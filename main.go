package main

import (
	"encoding/json"
	"flag"
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
	nsBindMountRoot  string
)

func parseArgs(allArgs []string) error {
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&networkSpec, "network", "", "")
	flagSet.StringVar(&cniPluginDir, "cniPluginDir", "", "")
	flagSet.StringVar(&cniConfigDir, "cniConfigDir", "", "")
	flagSet.StringVar(&ducatiSandboxDir, "ducatiSandboxDir", "", "")
	flagSet.StringVar(&daemonBaseURL, "daemonBaseURL", "", "")
	flagSet.StringVar(&nsBindMountRoot, "nsBindMountRoot", "", "")

	allArgs = allArgs[1:]
	err := flagSet.Parse(allArgs)
	if err != nil {
		os.Exit(1) // exit, error was already printed to stderr by flagSet.Parse
	}
	extraArgs := flagSet.Args()

	if len(extraArgs) > 0 {
		action = extraArgs[0]
		extraArgs = extraArgs[1:]
		err = flagSet.Parse(extraArgs)
		if err != nil {
			os.Exit(1) // exit, error already printed
		}
		extraArgs = flagSet.Args()
	}

	if len(extraArgs) > 0 {
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
	if nsBindMountRoot == "" {
		log.Fatalf("missing required flag 'nsBindMountRoot'")
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

	err = parseArgs(os.Args)
	if err != nil {
		log.Fatalf("arg parsing error: %s", err)
	}

	cniController := &controller.CNIController{
		PluginDir:      cniPluginDir,
		ConfigDir:      cniConfigDir,
		SandboxDirPath: ducatiSandboxDir,
		DaemonBaseURL:  daemonBaseURL,
	}

	mounter := &controller.Mounter{}

	manager := &controller.Manager{
		CNIController: cniController,
		Mounter:       mounter,
		BindMountRoot: nsBindMountRoot,
	}

	switch action {
	case "up":
		if containerState.Pid == 0 {
			log.Fatalf("missing pid")
		}
		err = manager.Up(containerState.Pid, handle, "some-spec-not-set")
		if err != nil {
			log.Fatalf("up failed: %s", err)
		}
	case "down":
		err = manager.Down(handle)
		if err != nil {
			log.Fatalf("down failed: %s", err)
		}
	default:
		log.Fatalf("action: %s is unrecognized", action)
	}
}
