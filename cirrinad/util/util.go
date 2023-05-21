package util

import (
	"encoding/json"
	"errors"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
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
		if strings.HasPrefix(inter.Name, "lo") {
			continue
		}
		if strings.HasPrefix(inter.Name, "bridge") {
			continue
		}
		if strings.HasPrefix(inter.Name, "tap") {
			continue
		}
		if strings.HasPrefix(inter.Name, "vmnet") {
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
