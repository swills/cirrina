package util

import (
	"bufio"
	"cirrina/cirrinad/config"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func PidExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err.Error() == "os: process already finished" {
		return false, nil
	}
	var errno syscall.Errno
	ok := errors.As(err, &errno)
	if !ok {
		return false, err
	}
	switch errno {
	case syscall.ESRCH:
		return false, nil
	case syscall.EPERM:
		return true, nil
	}
	return false, err
}

func FindChildPid(findPid uint32) (childPid uint32) {
	slog.Debug("FindChildPid finding child proc")
	pidString := strconv.FormatUint(uint64(findPid), 10)
	args := []string{"/bin/pgrep", "-P", pidString}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("FindChildPid error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	if err := cmd.Start(); err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	found := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		fl := len(textFields)
		if fl != 1 {
			slog.Debug("FindChildPid pgrep extra fields", "text", text)
		}
		tempPid1 := uint64(0)
		if !found {
			found = true
			tempPid1, err = strconv.ParseUint(textFields[0], 10, 32)
			if err != nil {
				slog.Error("FindChildPid error", "err", err)
				return 0
			}
			tempPid2 := uint32(tempPid1)
			childPid = tempPid2
		} else {
			slog.Debug("FindChildPid found too many child procs")
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("FindChildPid error", "err", err)
	}
	slog.Debug("FindChildPid returning childPid", "childPid", childPid)
	return childPid
}

func OSReadDir(root string) ([]string, error) {
	var files []string
	f, err := os.Open(root)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	_ = f.Close()
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

func ContainsStr(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func ContainsInt(elems []int, v int) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func GetFreeTCPPort(firstVncPort int, usedVncPorts []int) (port int, err error) {
	cmd := exec.Command("netstat", "-an", "--libxo", "json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	var result map[string]interface{}
	if err := json.NewDecoder(stdout).Decode(&result); err != nil {
		return 0, err
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("GetFreeTCPPort", "err", err)
		return 0, err
	}
	statistics, valid := result["statistics"].(map[string]interface{})
	if !valid {
		return 0, nil
	}
	sockets, valid := statistics["socket"].([]interface{})
	if !valid {
		return 0, errors.New("failed parsing netstat output - 1")
	}
	localListenPorts := make(map[int]struct{})
	for _, value := range sockets {
		socket, valid := value.(map[string]interface{})
		if !valid {
			continue
		}
		if socket["protocol"] == "tcp4" || socket["protocol"] == "tcp46" || socket["protocol"] == "tcp6" {
			state, valid := socket["tcp-state"].(string)
			if !valid {
				continue
			}
			realState := strings.TrimSpace(state)
			if realState == "LISTEN" {
				local, valid := socket["local"].(map[string]interface{})
				if !valid {
					continue
				}
				port, valid := local["port"].(interface{})
				if !valid {
					continue
				}
				p, valid := port.(string)
				if !valid {
					continue
				}
				portInt, err := strconv.Atoi(p)
				if err != nil {
					return 0, err
				}
				if _, exists := localListenPorts[portInt]; !exists {
					localListenPorts[portInt] = struct{}{}
				}
			}
		}
	}
	var uniqueLocalListenPorts []int
	for l := range localListenPorts {
		uniqueLocalListenPorts = append(uniqueLocalListenPorts, l)
	}
	sort.Slice(uniqueLocalListenPorts, func(i, j int) bool {
		return uniqueLocalListenPorts[i] < uniqueLocalListenPorts[j]
	})

	vncPort := firstVncPort
	for ; vncPort <= 65535; vncPort++ {
		if !ContainsInt(uniqueLocalListenPorts, vncPort) && !ContainsInt(usedVncPorts, vncPort) {
			break
		}
	}
	return vncPort, nil
}

func GetHostInterfaces() []string {
	var netDevs []string
	netInterfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	slog.Debug("GetHostInterfaces", "netInterfaces", netInterfaces)
	for _, inter := range netInterfaces {
		intGroups, err := GetIntGroups(inter.Name)
		if err != nil {
			slog.Error("failed to get interface groups", "err", err)
			return []string{}
		}
		if ContainsStr(intGroups, "cirrinad") {
			continue
		}
		if inter.HardwareAddr.String() == "" {
			continue
		}
		netDevs = append(netDevs, inter.Name)
	}
	return netDevs
}

func CopyFile(in, out string) (int64, error) {
	i, e := os.Open(in)
	if e != nil {
		return 0, e
	}
	defer func(i *os.File) {
		_ = i.Close()
	}(i)
	o, e := os.Create(out)
	if e != nil {
		return 0, e
	}
	defer func(o *os.File) {
		_ = o.Close()
	}(o)
	return o.ReadFrom(i)
}

func GetIntGroups(interfaceName string) (intGroups []string, err error) {
	cmd := exec.Command("/sbin/ifconfig", interfaceName)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []string{}, err
	}
	if err := cmd.Start(); err != nil {
		return []string{}, err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if !strings.HasPrefix(textFields[0], "groups:") {
			continue
		}
		fl := len(textFields)
		for f := 1; f < fl; f++ {
			intGroups = append(intGroups, textFields[f])
		}
	}
	if err := scanner.Err(); err != nil {
		return []string{}, err
	}
	return intGroups, nil
}

func ValidVmName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	var myRT = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return checkInRange(name, myRT)
}

func ValidDiskName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	var myRT = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return checkInRange(name, myRT)
}

func ValidIsoName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	var myRT = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002e, 1}, // - and .
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return checkInRange(name, myRT)
}

func ValidNicName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	var myRT = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return checkInRange(name, myRT)
}

func ValidSwitchName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	var myRT = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return checkInRange(name, myRT)
}

func checkInRange(name string, myRT *unicode.RangeTable) bool {
	for _, i := range name {
		if !unicode.In(i, myRT) {
			return false
		}
	}
	return true
}
