package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/db"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

var mainVersion = "unknown"

var cfgFile = "config.yml"

var shutdownHandlerRunning = false
var shutdownWaitGroup = sync.WaitGroup{}

func disableFlagSorting(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.InheritedFlags().SortFlags = false
}

func handleSigInfo() {
	var mem runtime.MemStats
	vm.LogAllVMStatus()
	runtime.ReadMemStats(&mem)
	slog.Debug("MemStats",
		"mem.Alloc", mem.Alloc,
		"mem.TotalAlloc", mem.TotalAlloc,
		"mem.HeapAlloc", mem.HeapAlloc,
		"mem.NumGC", mem.NumGC,
		"mem.Sys", mem.Sys,
	)
	runtime.GC()
	runtime.ReadMemStats(&mem)
	slog.Debug("MemStats",
		"mem.Alloc", mem.Alloc,
		"mem.TotalAlloc", mem.TotalAlloc,
		"mem.HeapAlloc", mem.HeapAlloc,
		"mem.NumGC", mem.NumGC,
		"mem.Sys", mem.Sys,
	)
}

func destroyPidFile() {
	pidFilePath, err := filepath.Abs(config.Config.Sys.PidFilePath)
	if err != nil {
		slog.Error("failed to get absolute path to pid file")
		os.Exit(1)
	}
	err = os.Remove(pidFilePath)
	if err != nil {
		slog.Error("failed removing leftover pid file")
		os.Exit(1)
	}
}

// write pid file, make sure it doesn't exist already, exit if it does
func writePidFile() {
	pidFilePath, err := filepath.Abs(config.Config.Sys.PidFilePath)
	if err != nil {
		slog.Error("failed to get absolute path to pid file")
		os.Exit(1)
	}
	slog.Debug("Checking pid file", "path", pidFilePath)
	_, err = os.Stat(pidFilePath)
	if err == nil {
		slog.Warn("pid file exists, checking pid")
		checkExistingPidFile(pidFilePath)
	}
	myPid := os.Getpid()

	var pidMode os.FileMode = 0x755
	err = os.WriteFile(pidFilePath, []byte(strconv.Itoa(myPid)), pidMode)
	if err != nil {
		slog.Error("failed writing pid file", "err", err)
		os.Exit(1)

		return
	}
}

func checkExistingPidFile(pidFilePath string) {
	existingPidFileContent, err := os.ReadFile(pidFilePath)
	if err != nil {
		slog.Error("pid file exists and unable to read it, please fix")
		os.Exit(1)
	}
	existingPid, err := strconv.Atoi(string(existingPidFileContent))
	if err != nil {
		slog.Error("failed getting existing pid")
		os.Exit(1)
	}
	slog.Debug("Checking pid", "pid", existingPid)
	procExists, err := util.PidExists(existingPid)
	if err != nil {
		slog.Error("failed checking existing pid")
		os.Exit(1)
	}
	if procExists {
		slog.Error("duplicate processes not allowed, please kill existing pid", "existingPid", existingPid)
		os.Exit(1)
	} else {
		slog.Warn("left over pid file detected, but process seems not to exist, deleting pid file")
		err := os.Remove(pidFilePath)
		if err != nil {
			slog.Error("failed removing leftover pid file, please fix")
			os.Exit(1)
		}
	}
}

func doDBMigrations() {
	util.ValidateDBConfig()

	// auto migration for meta (schema_version)
	db.AutoMigrate()

	// my custom migrations
	db.CustomMigrate()

	// gorm auto migrations
	disk.DBAutoMigrate()
	iso.DBAutoMigrate()
	vmnic.DBAutoMigrate()
	_switch.DBAutoMigrate()

	vm.DBAutoMigrate()

	requests.DBAutoMigrate()
}

func shutdownHandler() {
	if shutdownHandlerRunning {
		return
	}
	shutdownHandlerRunning = true
	vm.KillVMs()
	for {
		runningVMs := vm.GetRunningVMs()
		if runningVMs == 0 {
			break
		}
		slog.Info("waiting on running VM(s)", "count", runningVMs)
		time.Sleep(time.Second)
	}
	_switch.DestroyBridges()
	destroyPidFile()
	slog.Info("Exiting normally")
	shutdownWaitGroup.Done()
}

func sigHandler(signal os.Signal) {
	slog.Debug("got signal", "signal", signal)
	switch signal {
	case syscall.SIGINFO:
		handleSigInfo()
	case syscall.SIGINT:
		shutdownHandler()
	case syscall.SIGTERM:
		shutdownHandler()
	default:
		slog.Info("Ignoring signal", "signal", signal)
	}
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:          "cirrinad",
	Version:      mainVersion,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGINFO)
		signal.Notify(signals, os.Interrupt, syscall.SIGINT)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

		go func() {
			for {
				s := <-signals
				sigHandler(s)
			}
		}()

		var configAbsPath string
		configAbsPath, err = filepath.Abs(cfgFile)
		if err != nil {
			slog.Error("failed getting config file absolute path", "err", err)

			return err
		}

		var configPathExists bool
		configPathExists, err = util.PathExists(configAbsPath)
		if err != nil {
			slog.Error("error getting configAbsPath", "err", err)

			return err
		}
		if !configPathExists {
			return fmt.Errorf("config file %s not found", cfgFile)
		}

		err = viper.ReadInConfig()
		if err != nil {
			slog.Error("config reading failed", "err", err)

			return err
		}

		err = viper.UnmarshalExact(&config.Config, func(config *mapstructure.DecoderConfig) {
			config.TagName = "yaml"
			config.WeaklyTypedInput = true
		})
		if err != nil {
			slog.Error("config loading failed", "err", err)

			return err
		}

		validateLogConfig()

		logFile, err := os.OpenFile(config.Config.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			slog.Error("failed to open log file", err)

			return err
		}
		programLevel := new(slog.LevelVar) // Info by default
		logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: programLevel}))
		slog.SetDefault(logger) // any logging before this point is going to have to be at default level or higher
		switch strings.ToLower(config.Config.Log.Level) {
		case "debug":
			programLevel.Set(slog.LevelDebug)
		case "info":
			programLevel.Set(slog.LevelInfo)
		case "warn":
			programLevel.Set(slog.LevelWarn)
		case "error":
			programLevel.Set(slog.LevelError)
		default:
			programLevel.Set(slog.LevelInfo)
		}

		slog.Debug("Checking for existing proc")
		validatePidFilePathConfig()
		slog.Debug("Writing pid file")
		writePidFile()

		slog.Debug("Starting host validation")
		validateSystem()
		slog.Debug("Finished host validation")

		doDBMigrations()

		// check db contents
		validateDB()

		// code after this uses the database
		slog.Debug("Clean up starting")
		cleanupSystem()
		slog.Debug("Clean up complete")

		slog.Debug("Creating bridges")
		_switch.CreateBridges()
		slog.Info("Starting Daemon")

		go vm.AutoStartVMs()
		go rpcServer()
		go processRequests()

		shutdownWaitGroup.Add(1)
		shutdownWaitGroup.Wait()

		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false
	disableFlagSorting(rootCmd)

	rootCmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "C", cfgFile, "config file (default config.yml)",
	)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetDefault("sys.sudo", "/usr/local/bin/sudo")
	viper.SetDefault("sys.pidfilepath", "/var/run/cirrinad/cirrinad.pid")

	viper.SetDefault("db.path", "/var/db/cirrinad/cirrina.sqlite")

	// Maybe there could be default paths for disk paths?
	viper.SetDefault("disk.default.size", "1G")

	viper.SetDefault("log.path", "/var/log/cirrinad/cirrinad.log")
	viper.SetDefault("log.level", "info")

	viper.SetDefault("network.grpc.ip", "0.0.0.0")
	viper.SetDefault("network.grpc.port", 50051)
	// We use the "00:18:25" private OUI from
	// https://standards-oui.ieee.org/oui/oui.txt
	// as default, because why not? -- but you can customize it
	// you probably want to stick to the uni-cast (non-multicast) ones from that file
	// grep -i private oui.txt | grep -Ei base | grep -v '^.[13579BDF]' | grep -vi limited | grep -vi ltd
	// for more info, see:
	// https://en.wikipedia.org/wiki/MAC_address#Universal_vs._local_(U/L_bit)
	viper.SetDefault("network.mac.oui", "00:18:25")

	viper.SetDefault("rom.path", "/usr/local/share/uefi-firmware/BHYVE_UEFI.fd")
	viper.SetDefault("rom.vars.template", "/usr/local/share/uefi-firmware/BHYVE_UEFI_VARS.fd")

	viper.SetDefault("vnc.ip", "0.0.0.0")
	viper.SetDefault("vnc.port", 6900)

	viper.SetDefault("debug.ip", "0.0.0.0")
	viper.SetDefault("debug.port", 2828)

	viper.SetEnvPrefix("CIRRINAD")
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
}
