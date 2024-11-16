package epair

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func getAllEpair() ([]string, error) {
	var epairs []string

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/ifconfig",
		[]string{"-g", "epair"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("ifconfig error: %w", err)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)

		if len(textFields) != 1 {
			continue
		}

		if !strings.HasSuffix(textFields[0], "a") {
			continue
		}

		aPairName := strings.TrimSuffix(textFields[0], "a")
		epairs = append(epairs, aPairName)
	}

	return epairs, nil
}

func GetDummyEpairName() string {
	// highest if_bridge num
	epairNum := 32767

	ePairList, err := getAllEpair()
	if err != nil {
		return ""
	}

	for epairNum > 0 {
		epairName := "epair" + strconv.FormatInt(int64(epairNum), 10)
		if util.ContainsStr(ePairList, epairName) {
			epairNum--
		} else {
			return epairName
		}
	}

	return ""
}

func CreateEpair(name string) error {
	if name == "" {
		return errEpairNameEmpty
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name, "create", "group", "cirrinad"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name + "a", "up", "group", "cirrinad"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name + "b", "up", "group", "cirrinad"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}

	return nil
}

func DestroyEpair(name string) error {
	if name == "" {
		return errEpairNameEmpty
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name + "a", "destroy"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig error: %w", err)
	}

	return nil
}

func SetRateLimit(name string, rateIn uint64, rateOut uint64) error {
	var err error

	slog.Debug("setting rate limit on epair",
		"name", name,
		"rateIn", rateIn,
		"rateOut", rateOut,
	)

	err = NgCreatePipeWithRateLimit(name+"a", rateIn)
	if err != nil {
		slog.Error("error creating ng pipe with rate limit",
			"name", name,
			"rate", rateIn,
		)

		return fmt.Errorf("failed setting rate limit: %w", err)
	}

	err = NgCreatePipeWithRateLimit(name+"b", rateOut)
	if err != nil {
		slog.Error("error creating ng pipe with rate limit",
			"name", name,
			"rate", rateIn,
		)

		return fmt.Errorf("failed setting rate limit: %w", err)
	}

	return nil
}

func NgCreatePipeWithRateLimit(name string, rate uint64) error {
	var err error

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "mkpeer", name + ":", "pipe", "lower", "lower"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "name", name + ":lower", name + "_pipe"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", name + ":", name + "_pipe:", "upper", "upper"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	if rate != 0 {
		stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
			config.Config.Sys.Sudo,
			[]string{"/usr/sbin/ngctl", "msg", name + "_pipe:", "setcfg",
				"{", "upstream={", "bandwidth=" + strconv.FormatInt(int64(rate), 10), "fifo=1", "}", "}",
			},
		)
		if err != nil {
			slog.Error("ngctl error",
				"stdOutBytes", stdOutBytes,
				"stdErrBytes", stdErrBytes,
				"returnCode", returnCode,
				"err", err,
			)

			return fmt.Errorf("ngctl error: %w", err)
		}
	}

	return nil
}

func NgDestroyPipe(name string) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "shutdown", name + "_pipe" + ":"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	return nil
}
