package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"

	exec "golang.org/x/sys/execabs"
	"golang.org/x/sys/unix"

	"cirrina/cirrinad/config"
)

func PathExists(path string) (bool, error) {
	statInfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || statInfo == nil {
			return false, nil
		}
		return false, err
	}
	slog.Debug("PathExists", "path", path, "statInfo", statInfo)
	return true, nil
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
	if errors.Is(errno, syscall.ESRCH) {
		return false, nil
	}
	if errors.Is(errno, syscall.EPERM) {
		return true, nil
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

func captureReader(r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}

// runCommandAndCaptureOutput does what it says on the tin. maybe I should make a version that takes a string to pass
// in on standard input. also maybe I should make this public here. perhaps pointer args would be better
func runCommandAndCaptureOutput(cmdName string, cmdArgs []string) ([]byte, error) {
	var outResult []byte
	var errResult []byte
	var errStdout, errStderr error

	cmd := exec.Command(cmdName, cmdArgs...)
	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, err
	}
	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		return []byte{}, err
	}
	if err := cmd.Start(); err != nil {
		return []byte{}, err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		outResult, errStdout = captureReader(stdoutIn)
		wg.Done()
	}()

	errResult, errStderr = captureReader(stderrIn)

	wg.Wait()

	if errStdout != nil {
		return []byte{}, errStderr
	}
	if errStderr != nil {
		return []byte{}, errStderr
	}
	if len(errResult) > 0 {
		return []byte{}, errors.New(string(errResult))
	}
	if err := cmd.Wait(); err != nil {
		return []byte{}, err
	}

	return outResult, nil
}

func getNetstatJsonOutput() ([]byte, error) {
	return runCommandAndCaptureOutput("/usr/bin/netstat", []string{"-an", "--libxo", "json"})
}

func parseNetstatSocket(socket map[string]interface{}) (int, error) {
	var portInt int
	var err error

	if socket["protocol"] != "tcp4" && socket["protocol"] != "tcp46" && socket["protocol"] != "tcp6" {
		return 0, errors.New("not a tcp socket")
	}
	state, valid := socket["tcp-state"].(string)
	if !valid {
		return 0, errors.New("missing tcp-stat")
	}
	realState := strings.TrimSpace(state)
	if realState != "LISTEN" {
		return 0, errors.New("port is not a listen port")
	}
	local, valid := socket["local"].(map[string]interface{})
	if !valid {
		return 0, errors.New("not a listen socket")
	}
	port, valid := local["port"]
	if !valid {
		return 0, errors.New("tcp port not found")
	}
	p, valid := port.(string)
	if !valid {
		return 0, errors.New("tcp port not parsable")
	}
	portInt, err = strconv.Atoi(p)
	if err != nil {
		return 0, errors.New("tcp port failed to convert to int")
	}
	return portInt, nil
}

func parseNetstatJsonOutput(netstatOutput []byte) ([]int, error) {
	var result map[string]interface{}

	err := json.Unmarshal(netstatOutput, &result)
	if err != nil {
		return nil, err
	}
	statistics, valid := result["statistics"].(map[string]interface{})
	if !valid {
		return nil, errors.New("failed parsing output, statistics not found")
	}
	sockets, valid := statistics["socket"].([]interface{})
	if !valid {
		return nil, errors.New("failed parsing output, socket not found")
	}

	var localPortList []int
	for _, value := range sockets {
		socket, valid := value.(map[string]interface{})
		if !valid {
			continue
		}
		portInt, err := parseNetstatSocket(socket)
		if err != nil {
			continue
		}
		if !ContainsInt(localPortList, portInt) {
			localPortList = append(localPortList, portInt)
		}
	}

	return localPortList, nil
}

func GetFreeTCPPort(firstVncPort int, usedVncPorts []int) (port int, err error) {
	// get and parse netstat output
	netstatJson, err := getNetstatJsonOutput()
	if err != nil {
		return 0, err
	}
	uniqueLocalListenPorts, err := parseNetstatJsonOutput(netstatJson)
	if err != nil {
		return 0, err
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

func MacIsBroadcast(macAddress string) (bool, error) {
	newMac, err := net.ParseMAC(macAddress)
	if err != nil {
		return false, errors.New("invalid MAC address")
	}
	if bytes.Equal(newMac, []byte{255, 255, 255, 255, 255, 255}) {
		return true, nil
	}
	return false, nil
}

func MacIsMulticast(macAddress string) (bool, error) {
	newMac, err := net.ParseMAC(macAddress)
	if err != nil {
		return false, errors.New("invalid MAC address")
	}
	// https://cgit.freebsd.org/src/tree/usr.sbin/bhyve/net_utils.c?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n56
	// https://cgit.freebsd.org/src/tree/sys/net/ethernet.h?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n74
	if newMac[0]&0x01 == 1 {
		return true, nil
	}
	return false, nil
}

func IsValidIP(ipAddress string) bool {
	parsedIp := net.ParseIP(ipAddress)
	return parsedIp != nil
}

func IsValidTcpPort(tcpPort uint) bool {
	if tcpPort < 1 || tcpPort > 65535 {
		return false
	}
	return true
}

func ModeIsSuid(mode fs.FileMode) bool {
	return mode&fs.ModeSetuid != 0
}

// func ModeIsWriteOwner(mode os.FileMode) bool {
// 	return mode&0200 != 0
// }

func ModeIsExecOther(mode os.FileMode) bool {
	return mode&0001 != 0
}

func GetMyUidGid() (uid uint32, gid uint32, err error) {
	myUser, err := user.Current()
	if err != nil {
		return 0, 0, err
	}
	myUid, err := strconv.Atoi(myUser.Uid)
	if err != nil {
		return 0, 0, err
	}
	myGid, err := strconv.Atoi(myUser.Gid)
	if err != nil {
		return 0, 0, err
	}
	u := uint32(myUid)
	g := uint32(myGid)
	return u, g, nil
}

func ValidateDbConfig() {
	dbFilePath, err := filepath.Abs(config.Config.DB.Path)
	if err != nil {
		slog.Error("failed to get absolute path to database")
		os.Exit(1)
	}
	dbFilePathInfo, err := os.Stat(dbFilePath)
	// db file will be created if it does not exist
	if err == nil {
		// however, if the path specified for the db does exist, it must not be a directory
		if dbFilePathInfo.IsDir() {
			slog.Error("database path is a directory, please reconfigure to point to a file", "dbFilePath", dbFilePath)
			os.Exit(1)
		}
	}
	dbDir := filepath.Dir(config.Config.DB.Path)
	if unix.Access(dbDir, unix.W_OK) != nil {
		errM := fmt.Sprintf("db dir %s not writable", dbDir)
		slog.Error(errM)
		os.Exit(1)
	}
}

func ParseDiskSize(size string) (sizeBytes uint64, err error) {
	var t string
	var n uint
	var m uint64
	switch {
	case strings.HasSuffix(size, "k"):
		t = strings.TrimSuffix(size, "k")
		m = 1024
	case strings.HasSuffix(size, "K"):
		t = strings.TrimSuffix(size, "K")
		m = 1024
	case strings.HasSuffix(size, "m"):
		t = strings.TrimSuffix(size, "m")
		m = 1024 * 1024
	case strings.HasSuffix(size, "M"):
		t = strings.TrimSuffix(size, "M")
		m = 1024 * 1024
	case strings.HasSuffix(size, "g"):
		t = strings.TrimSuffix(size, "g")
		m = 1024 * 1024 * 1024
	case strings.HasSuffix(size, "G"):
		t = strings.TrimSuffix(size, "G")
		m = 1024 * 1024 * 1024
	case strings.HasSuffix(size, "t"):
		t = strings.TrimSuffix(size, "t")
		m = 1024 * 1024 * 1024 * 1024
	case strings.HasSuffix(size, "T"):
		t = strings.TrimSuffix(size, "T")
		m = 1024 * 1024 * 1024 * 1024
	case strings.HasSuffix(size, "b"):
		t = strings.TrimSuffix(size, "b")
		m = 1024 * 1024 * 1024 * 1024
	default:
		t = size
		m = 1
	}
	nu, err := strconv.Atoi(t)
	if err != nil {
		return 0, err
	}
	if nu < 1 {
		return 0, fmt.Errorf("invalid disk size %s", size)
	}
	n = uint(nu)
	r := uint64(n) * m
	return r, nil
}

func GetHostMaxVmCpus() (uint16, error) {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer

	checkCmd := exec.Command("/sbin/sysctl", "-n", "hw.vmm.maxcpu")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		slog.Error("Failed getting max vm cpus", "command", checkCmd.String(), "err", err.Error())
		return 0, err
	}
	maxCpuStr := strings.TrimSpace(outBytes.String())
	maxCpu, err := strconv.Atoi(maxCpuStr)
	if err != nil {
		slog.Error("Failed converting max cpus to int", "err", err.Error())
		return 0, err
	}
	if maxCpu <= 0 || maxCpu >= math.MaxUint16 {
		slog.Error("Failed invalid max cpus", "maxCpu", maxCpu)
		return 0, err
	}
	return uint16(maxCpu), nil
}
