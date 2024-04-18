package epair

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func getAllEpair() ([]string, error) {
	var err error
	var epairs []string
	cmd := exec.Command("/sbin/ifconfig", "-g", "epair")
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)
	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return []string{}, fmt.Errorf("error running ifconfig: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return []string{}, fmt.Errorf("error running ifconfig: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 1 {
			continue
		}
		if strings.HasSuffix(textFields[0], "b") {
			continue
		}
		aPairName := strings.TrimSuffix(textFields[0], "a")
		epairs = append(epairs, aPairName)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("error scanning ifconfig output", "err", err)

		return []string{}, fmt.Errorf("error parsing ifconfig output: %w", err)
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
		epairName := "epair" + strconv.Itoa(epairNum)
		if util.ContainsStr(ePairList, epairName) {
			epairNum--
		} else {
			return epairName
		}
	}

	return ""
}

func CreateEpair(name string) error {
	var err error
	if name == "" {
		return errEpairNameEmpty
	}
	args := []string{"/sbin/ifconfig", name, "create", "group", "cirrinad"}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to create epair", "name", name, "err", err)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}
	args = []string{"/sbin/ifconfig", name + "a", "up", "group", "cirrinad"}
	cmd = exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to up epair", "name", name+"a", "err", err)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}
	args = []string{"/sbin/ifconfig", name + "b", "up", "group", "cirrinad"}
	cmd = exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to up epair", "name", name+"b", "err", err)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}

	return nil
}

func DestroyEpair(name string) error {
	var err error
	if name == "" {
		return errEpairNameEmpty
	}
	args := []string{"/sbin/ifconfig", name + "a", "destroy"}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to destroy epair", "name", name+"a", "err", err)

		return fmt.Errorf("failed running ifconfig: %w", err)
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
	slog.Debug("creating ng pipe",
		"name", name,
		"rate", rate,
	)

	cmd := exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "mkpeer", name+":", "pipe", "lower", "lower")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error ng pipe peer",
			"name", name,
			"err", err,
		)

		return fmt.Errorf("failed running ngctl: %w", err)
	}

	cmd = exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "name", name+":lower", name+"_pipe")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl setting pipe name",
			"name", name,
			"err", err,
		)

		return fmt.Errorf("failed running ngctl: %w", err)
	}

	cmd = exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "connect", name+":", name+"_pipe:", "upper", "upper")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl setting pipe name",
			"name", name,
			"err", err,
		)

		return fmt.Errorf("failed running ngctl: %w", err)
	}

	if rate != 0 {
		cmd = exec.Command(config.Config.Sys.Sudo,
			"/usr/sbin/ngctl", "msg",
			name+"_pipe:",
			"setcfg", "{", "upstream={", "bandwidth="+strconv.Itoa(int(rate)), "fifo=1", "}", "}",
		)
		err = cmd.Run()
		if err != nil {
			slog.Error("ngctl setting pipe rate",
				"name", name,
				"rate", rate,
				"err", err,
			)

			return fmt.Errorf("failed running ngctl: %w", err)
		}
	}

	return nil
}

func NgDestroyPipe(name string) error {
	var err error
	cmd := exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "shutdown", name+"_pipe"+":")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error ng pipe peer",
			"name", name,
			"err", err,
		)

		return fmt.Errorf("failed running ngctl: %w", err)
	}

	return nil
}
