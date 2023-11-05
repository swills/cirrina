package epair

import (
	"bufio"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	exec "golang.org/x/sys/execabs"
	"strconv"
	"strings"
)

func getAllEpair() (epairs []string, err error) {
	var r []string
	cmd := exec.Command("/sbin/ifconfig", "-g", "epair")
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
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
		r = append(r, aPairName)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
	return r, nil
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
			epairNum = epairNum - 1
		} else {
			return epairName
		}
	}

	return ""
}

func CreateEpair(name string) (err error) {
	if name == "" {
		return errors.New("empty epair name")
	}
	args := []string{"/sbin/ifconfig", name, "create", "group", "cirrinad"}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to create epair", "name", name, "err", err)
		return err
	}
	args = []string{"/sbin/ifconfig", name + "a", "up", "group", "cirrinad"}
	cmd = exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to up epair", "name", name+"a", "err", err)
		return err
	}
	args = []string{"/sbin/ifconfig", name + "b", "up", "group", "cirrinad"}
	cmd = exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to up epair", "name", name+"b", "err", err)
		return err
	}
	return nil
}

func DestroyEpair(name string) (err error) {
	if name == "" {
		return errors.New("empty epair name")
	}
	args := []string{"/sbin/ifconfig", name + "a", "destroy"}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to destroy epair", "name", name+"a", "err", err)
		return err
	}
	return nil
}

func SetRateLimit(name string, rateIn uint64, rateOut uint64) (err error) {
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
		return err
	}
	err = NgCreatePipeWithRateLimit(name+"b", rateOut)
	if err != nil {
		slog.Error("error creating ng pipe with rate limit",
			"name", name,
			"rate", rateIn,
		)
		return err
	}
	return nil
}

func NgCreatePipeWithRateLimit(name string, rate uint64) (err error) {
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
		return err
	}

	cmd = exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "name", name+":lower", name+"_pipe")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl setting pipe name",
			"name", name,
			"err", err,
		)
		return err
	}

	cmd = exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "connect", name+":", name+"_pipe:", "upper", "upper")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl setting pipe name",
			"name", name,
			"err", err,
		)
		return err
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
			return err
		}
	}

	return nil
}

func NgDestroyPipe(name string) (err error) {
	cmd := exec.Command(config.Config.Sys.Sudo,
		"/usr/sbin/ngctl", "shutdown", name+"_pipe"+":")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error ng pipe peer",
			"name", name,
			"err", err,
		)
		return err
	}

	return nil
}
