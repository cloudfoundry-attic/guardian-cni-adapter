package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"lib/marshal"
	"log"
	"os"
	"path/filepath"

	GardenClient "github.com/cloudfoundry-incubator/garden/client"
	GardenConnection "github.com/cloudfoundry-incubator/garden/client/connection"
	"github.com/cloudfoundry-incubator/guardian-cni-adapter/controller"
	"github.com/cloudfoundry-incubator/guardian-cni-adapter/lookup"
)

type Config struct {
	CniPluginDir string `json:"cni_plugin_dir"`
	CniConfigDir string `json:"cni_config_dir"`
	BindMountDir string `json:"bind_mount_dir"`
	LogDir       string `json:"log_dir"`
	Garden       struct {
		Address string `json:"address"`
		Network string `json:"network"`
	} `json:"garden"`
}

var (
	action      string
	handle      string
	networkSpec string
	config      Config
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

func parseConfig(configFilePath string) error {
	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("reading config file: %s", err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return fmt.Errorf("parsing config (%s): %s", configFilePath, err)
	}

	if config.LogDir == "" {
		return fmt.Errorf("missing required config 'log_dir'")
	}

	if config.CniPluginDir == "" {
		return fmt.Errorf("missing required config 'cni_plugin_dir'")
	}

	if config.CniConfigDir == "" {
		return fmt.Errorf("missing required config 'cni_config_dir'")
	}

	if config.BindMountDir == "" {
		return fmt.Errorf("missing required config 'bind_mount_dir'")
	}

	return nil
}

func parseArgs(allArgs []string) error {
	var gardenNetworkSpec, configFilePath string
	var externalNetworkSpec string

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&action, "action", "", "")
	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&gardenNetworkSpec, "network", "", "")
	flagSet.StringVar(&externalNetworkSpec, "external-network", "", "")
	flagSet.StringVar(&configFilePath, "configFile", "", "")

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

	if configFilePath == "" {
		return fmt.Errorf("missing required flag 'configFile'")
	}

	if err = parseConfig(configFilePath); err != nil {
		return err
	}

	if err = setupLogging(config.LogDir, handle); err != nil {
		return err
	}

	if action == "" {
		return fmt.Errorf("missing required flag 'action'")
	}

	log.Printf("parsed: %+v\n", allArgs)
	log.Printf("adapter config: %+v\n", config)

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
		PluginDir: config.CniPluginDir,
		ConfigDir: config.CniConfigDir,
	}

	mounter := &controller.Mounter{}

	manager := &controller.Manager{
		CNIController: cniController,
		Mounter:       mounter,
		BindMountRoot: config.BindMountDir,
	}

	gardenClient := GardenClient.New(GardenConnection.New(config.Garden.Network, config.Garden.Address))
	lookupClient := lookup.Client{
		GardenClient: gardenClient,
		Marshaler:    marshal.MarshalFunc(json.Marshal),
	}

	switch action {
	case "up":
		networkSpec, err := lookupClient.GetNetworkSpec(handle)
		if err != nil {
			log.Fatalf("get network spec: %s", err)
		}

		err = manager.Up(containerState.Pid, handle, networkSpec)
		if err != nil {
			log.Fatalf("up failed: %s", err)
		}
	case "down":
		networkSpec, err := lookupClient.GetNetworkSpec(handle)
		if err != nil {
			log.Fatalf("get network spec: %s", err)
		}

		err = manager.Down(handle, networkSpec)
		if err != nil {
			log.Fatalf("down failed: %s", err)
		}
	default:
		log.Fatalf("action: %s is unrecognized", action)
	}
}
