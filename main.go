package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

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
	logDir           string
)

func setupLogging(logDir, handle string) error {
	if logDir == "" {
		return nil
	}

	if err := os.MkdirAll(logDir, 0644); err != nil {
		return fmt.Errorf("unable to create log dir %q: %s", logDir, err)
	}

	logFilePath := filepath.Join(logDir, handle+".log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("unable to create log file %q: %s", logFilePath, err)
	}
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	log.Printf("started logging to %s\n", logFilePath)
	return nil
}

func parseArgs(allArgs []string) error {
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&action, "action", "", "")
	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&networkSpec, "network", "", "")
	flagSet.StringVar(&cniPluginDir, "cniPluginDir", "", "")
	flagSet.StringVar(&cniConfigDir, "cniConfigDir", "", "")
	flagSet.StringVar(&ducatiSandboxDir, "ducatiSandboxDir", "", "")
	flagSet.StringVar(&daemonBaseURL, "daemonBaseURL", "", "")
	flagSet.StringVar(&nsBindMountRoot, "nsBindMountRoot", "", "")
	flagSet.StringVar(&logDir, "logDir", "", "")

	err := flagSet.Parse(allArgs[1:])
	if err != nil {
		return err
	}
	if len(flagSet.Args()) > 0 {
		return fmt.Errorf("unexpected extra args: %+v", flagSet.Args())
	}

	if handle == "" {
		return fmt.Errorf("missing required flag 'handle'")
	}

	if err := setupLogging(logDir, handle); err != nil {
		return fmt.Errorf("failed setting up logging: %s", err)
	}

	if action == "" {
		return fmt.Errorf("missing required flag 'action'")
	}
	if cniPluginDir == "" {
		return fmt.Errorf("missing required flag 'cniPluginDir'")
	}
	if cniConfigDir == "" {
		return fmt.Errorf("missing required flag 'cniConfigDir'")
	}
	if ducatiSandboxDir == "" {
		return fmt.Errorf("missing required flag 'ducatiSandboxDir'")
	}
	if daemonBaseURL == "" {
		return fmt.Errorf("missing required flag 'daemonBaseURL'")
	}
	if nsBindMountRoot == "" {
		return fmt.Errorf("missing required flag 'nsBindMountRoot'")
	}
	if action == "up" && networkSpec == "" {
		return fmt.Errorf("missing required flag 'network'")
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
		err = manager.Up(containerState.Pid, handle, networkSpec)
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
