package util

import (
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
	"regexp"
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

// allow testing
var execute = exec.Command
var getHostMaxVMCpusFunc = GetHostMaxVMCpus
var getIntGroupsFunc = GetIntGroups
var netInterfacesFunc = net.Interfaces
var osOpenFunc = os.Open

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("failed checking path exists: %w", err)
	}

	return true, nil
}

func PidExists(pid int) (bool, error) {
	// TODO get sysctl kern.pid_max_limit and/or kern.pid_max and compare
	if pid <= 0 {
		return false, errInvalidPid
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("failed checking pid exists: %w", err)
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrProcessDone) {
		return false, nil
	}

	var errno syscall.Errno

	ok := errors.As(err, &errno)
	if !ok {
		return false, fmt.Errorf("failed checking pid exists: %w", err)
	}

	if errors.Is(errno, syscall.ESRCH) {
		return false, nil
	}

	if errors.Is(errno, syscall.EPERM) {
		return true, nil
	}

	return false, fmt.Errorf("failed checking pid exists: %w", err)
}

func OSReadDir(path string) ([]string, error) {
	var files []string

	pathFile, err := osOpenFunc(path)
	if err != nil {
		return []string{}, fmt.Errorf("failed reading OS dir: %w", err)
	}

	fileInfo, err := pathFile.Readdir(-1)
	_ = pathFile.Close()

	if err != nil {
		return []string{}, fmt.Errorf("failed reading OS dir: %w", err)
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

func captureReader(ioReader io.Reader) ([]byte, error) {
	var out []byte

	buf := make([]byte, 1024)

	for {
		n, err := ioReader.Read(buf)
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

// RunCmd execute a system command and return stdout, stderr, return code and any internal errors
// encountered running the command
func RunCmd(cmdName string, cmdArgs []string) ([]byte, []byte, int, error) {
	var err error

	var outResult []byte

	var errResult []byte

	var errStdout, errStderr error

	slog.Debug("RunCmd running",
		"cmdName", cmdName,
		"cmdArgs", cmdArgs,
	)

	cmd := execute(cmdName, cmdArgs...)

	stdOutReader, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, []byte{}, 0, fmt.Errorf("error running command: %w", err)
	}

	stdErrReader, err := cmd.StderrPipe()
	if err != nil {
		return []byte{}, []byte{}, 0, fmt.Errorf("error running command: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return []byte{}, []byte{}, 0, fmt.Errorf("error running command: %w", err)
	}

	var runCmdWaitGroup sync.WaitGroup

	runCmdWaitGroup.Add(1)

	go func() {
		outResult, errStdout = captureReader(stdOutReader)

		runCmdWaitGroup.Done()
	}()

	errResult, errStderr = captureReader(stdErrReader)

	runCmdWaitGroup.Wait()

	if errStdout != nil {
		return []byte{}, []byte{}, 0, errStdout
	}

	if errStderr != nil {
		return []byte{}, []byte{}, 0, errStderr
	}

	returnCode := 0

	err = cmd.Wait()
	if err != nil {
		var exiterr *exec.ExitError
		if errors.As(err, &exiterr) {
			returnCode = cmd.ProcessState.ExitCode()
		}

		return outResult, errResult, returnCode, fmt.Errorf("error running command: %w", err)
	}

	return outResult, errResult, returnCode, nil
}

func parseNetstatSocket(socket map[string]interface{}) (int, error) {
	var portInt int

	var err error

	if socket["protocol"] != "tcp4" && socket["protocol"] != "tcp46" && socket["protocol"] != "tcp6" {
		return 0, errNoTCPSocket
	}

	state, valid := socket["tcp-state"].(string)
	if !valid {
		return 0, errMissingTCPStat
	}

	realState := strings.TrimSpace(state)
	if realState != "LISTEN" {
		return 0, errNoListenPort
	}

	local, valid := socket["local"].(map[string]interface{})
	if !valid {
		return 0, errNoListenSocket
	}

	port, valid := local["port"]
	if !valid {
		return 0, errPortNotFound
	}

	p, valid := port.(string)
	if !valid {
		return 0, errPortNotParsable
	}

	portInt, err = strconv.Atoi(p)
	if err != nil {
		return 0, errInvalidPort
	}

	return portInt, nil
}

func parseNetstatJSONOutput(netstatOutput []byte) ([]int, error) {
	var result map[string]interface{}

	err := json.Unmarshal(netstatOutput, &result)
	if err != nil {
		return nil, fmt.Errorf("failed parsing netstat json output: %w", err)
	}

	statistics, valid := result["statistics"].(map[string]interface{})
	if !valid {
		return nil, errFailedParsing
	}

	sockets, valid := statistics["socket"].([]interface{})
	if !valid {
		return nil, errSocketNotFound
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

func GetFreeTCPPort(firstVncPort int, usedVncPorts []int) (int, error) {
	var err error
	// get and parse netstat output
	stdOutBytes, stdErrBytes, rc, err := RunCmd("/usr/bin/netstat", []string{"-an", "--libxo", "json"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("error running command", "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)

		return 0, fmt.Errorf("error running sysctl: stderr: %s, rc: %d, err: %w", string(stdErrBytes), rc, err)
	}

	uniqueLocalListenPorts, err := parseNetstatJSONOutput(stdOutBytes)
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

	netInterfaces, _ := netInterfacesFunc()

	for _, inter := range netInterfaces {
		intGroups, err := getIntGroupsFunc(inter.Name)
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
	inFile, err := os.Open(in)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer func(i *os.File) {
		_ = i.Close()
	}(inFile)

	outFile, err := os.Create(out)
	if err != nil {
		return 0, fmt.Errorf("error creating file: %w", err)
	}
	defer func(o *os.File) {
		_ = o.Close()
	}(outFile)

	n, err := outFile.ReadFrom(inFile)
	if err != nil {
		return n, fmt.Errorf("error copying file: %w", err)
	}

	return n, nil
}

// GetIntGroups returns the list of groups the interface is in
func GetIntGroups(interfaceName string) ([]string, error) {
	var intGroups []string

	stdOutBytes, stdErrBytes, rc, err := RunCmd("/sbin/ifconfig", []string{interfaceName})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("error running command", "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)

		return []string{}, fmt.Errorf("error running ifconfig: stderr: %s, rc: %d, err: %w", string(stdErrBytes), rc, err)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		textFields := strings.Fields(line)
		if len(textFields) < 1 || !strings.HasPrefix(textFields[0], "groups:") {
			continue
		}

		fl := len(textFields)
		for f := 1; f < fl; f++ {
			intGroups = append(intGroups, textFields[f])
		}
	}

	return intGroups, nil
}

// ValidVMName checks if a name is a valid name for a VM
func ValidVMName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	myRT := &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return CheckInRange(name, myRT)
}

// ValidDiskName checks if a name is a valid name for a disk
func ValidDiskName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	myRT := &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002e, 1}, // - and .
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	inRange := CheckInRange(name, myRT)
	if !inRange {
		return false
	}

	matchesDoubleDot, _ := regexp.MatchString(`\.\.`, name)

	if matchesDoubleDot {
		return false
	}

	matchesLeadingDot, _ := regexp.MatchString(`^\.`, name)

	return !matchesLeadingDot
}

// ValidIsoName checks if a name is a valid name for an ISO
func ValidIsoName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	myRT := &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002e, 1}, // - and .
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return CheckInRange(name, myRT)
}

// ValidNicName check if a name is valid for a NIC
func ValidNicName(name string) bool {
	if name == "" {
		return false
	}

	// values must be kept sorted
	myRT := &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	return CheckInRange(name, myRT)
}

// CheckInRange check if a name contains any characters not in the unicode range table provided
func CheckInRange(name string, myRT *unicode.RangeTable) bool {
	for _, i := range name {
		if !unicode.In(i, myRT) {
			return false
		}
	}

	return true
}

// MacIsBroadcast check if a MAC address is a broadcast MAC
func MacIsBroadcast(macAddress string) (bool, error) {
	newMac, err := net.ParseMAC(macAddress)
	if err != nil {
		return false, errInvalidMac
	}

	if len(newMac.String()) != 17 {
		return false, errInvalidMac
	}

	if bytes.Equal(newMac, []byte{255, 255, 255, 255, 255, 255}) {
		return true, nil
	}

	return false, nil
}

// MacIsMulticast check if a MAC is a multicast MAC
func MacIsMulticast(macAddress string) (bool, error) {
	newMac, err := net.ParseMAC(macAddress)
	if err != nil {
		return false, errInvalidMac
	}

	if len(newMac.String()) != 17 {
		return false, errInvalidMac
	}
	// https://cgit.freebsd.org/src/tree/usr.sbin/bhyve/net_utils.c?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n56
	// https://cgit.freebsd.org/src/tree/sys/net/ethernet.h?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n74
	if newMac[0]&0x01 == 1 {
		return true, nil
	}

	return false, nil
}

func IsValidIP(ipAddress string) bool {
	parsedIP := net.ParseIP(ipAddress)

	return parsedIP != nil
}

// IsValidTCPPort check if a number is a valid TCP port
func IsValidTCPPort(tcpPort uint) bool {
	return tcpPort <= 65535
}

func ModeIsSuid(mode fs.FileMode) bool {
	return mode&fs.ModeSetuid != 0
}

// func ModeIsWriteOwner(mode os.FileMode) bool {
// 	return mode&0200 != 0
// }

func ModeIsExecOther(mode os.FileMode) bool {
	return mode&0o001 != 0
}

func GetMyUIDGID() (uint32, uint32, error) {
	var err error

	var myUser *user.User

	myUser, err = user.Current()
	if err != nil {
		return 0, 0, fmt.Errorf("error getting current user: %w", err)
	}

	if myUser == nil {
		return 0, 0, errUserNotFound
	}

	var myUID int

	myUID, err = strconv.Atoi(myUser.Uid)
	if err != nil || myUID < 0 {
		return 0, 0, fmt.Errorf("error parsing UID: %w", err)
	}

	var myGID int

	myGID, err = strconv.Atoi(myUser.Gid)
	if err != nil || myGID < 0 {
		return 0, 0, fmt.Errorf("error parsing GID: %w", err)
	}

	return uint32(myUID), uint32(myGID), nil
}

func ValidateDBConfig() {
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

func ParseDiskSize(diskSize string) (uint64, error) {
	var err error

	var diskSizeNum uint64

	trimmedSize, multiplier := parseDiskSizeSuffix(diskSize)

	diskSizeNum, err = strconv.ParseUint(trimmedSize, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed parsing disk size: %w", err)
	}

	if multiplyWillOverflow(diskSizeNum, multiplier) {
		return 0, errInvalidDiskSize
	}

	finalSize := diskSizeNum * multiplier

	// limit disks to min 512 bytes, max 128TB
	if finalSize < 512 || finalSize > 1024*1024*1024*1024*128 {
		return 0, errInvalidDiskSize
	}

	return finalSize, nil
}

func parseDiskSizeSuffix(diskSize string) (string, uint64) {
	var trimmedSize string

	var multiplier uint64

	switch {
	case strings.HasSuffix(diskSize, "b"):
		trimmedSize = strings.TrimSuffix(diskSize, "b")
		multiplier = 1
	case strings.HasSuffix(diskSize, "B"):
		trimmedSize = strings.TrimSuffix(diskSize, "B")
		multiplier = 1
	case strings.HasSuffix(diskSize, "k"):
		trimmedSize = strings.TrimSuffix(diskSize, "k")
		multiplier = 1024
	case strings.HasSuffix(diskSize, "K"):
		trimmedSize = strings.TrimSuffix(diskSize, "K")
		multiplier = 1024
	case strings.HasSuffix(diskSize, "m"):
		trimmedSize = strings.TrimSuffix(diskSize, "m")
		multiplier = 1024 * 1024
	case strings.HasSuffix(diskSize, "M"):
		trimmedSize = strings.TrimSuffix(diskSize, "M")
		multiplier = 1024 * 1024
	case strings.HasSuffix(diskSize, "g"):
		trimmedSize = strings.TrimSuffix(diskSize, "g")
		multiplier = 1024 * 1024 * 1024
	case strings.HasSuffix(diskSize, "G"):
		trimmedSize = strings.TrimSuffix(diskSize, "G")
		multiplier = 1024 * 1024 * 1024
	case strings.HasSuffix(diskSize, "t"):
		trimmedSize = strings.TrimSuffix(diskSize, "t")
		multiplier = 1024 * 1024 * 1024 * 1024
	case strings.HasSuffix(diskSize, "T"):
		trimmedSize = strings.TrimSuffix(diskSize, "T")
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		trimmedSize = diskSize
		multiplier = 1
	}

	return trimmedSize, multiplier
}

func GetHostMaxVMCpus() (uint16, error) {
	stdOutBytes, stdErrBytes, rc, err := RunCmd("/sbin/sysctl", []string{"-n", "hw.vmm.maxcpu"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		return 0, fmt.Errorf("error running sysctl: stderr: %s, rc: %d, err: %w", string(stdErrBytes), rc, err)
	}

	maxCPUStr := strings.TrimSpace(string(stdOutBytes))

	maxCPU, err := strconv.Atoi(maxCPUStr)
	if err != nil {
		slog.Error("Failed converting max cpus to int", "err", err.Error())

		return 0, fmt.Errorf("error parsing cpu count: %w", err)
	}

	if maxCPU <= 0 || maxCPU >= math.MaxUint16 {
		slog.Error("Failed invalid max cpus", "maxCPU", maxCPU)

		return 0, errInvalidNumCPUs
	}

	return uint16(maxCPU), nil
}

func multiplyWillOverflow(xVal, yVal uint64) bool {
	if xVal <= 1 || yVal <= 1 {
		return false
	}

	d := xVal * yVal

	return d/yVal != xVal
}

func NumCpusValid(numCpus uint16) bool {
	hostCpus, err := getHostMaxVMCpusFunc()
	if err != nil {
		slog.Error("error getting number of host cpus", "err", err)

		return false
	}

	if numCpus > hostCpus {
		return false
	}

	return true
}

// SetupTestCmd replaces the executing command to the fake function implemented by Go lang
// from https://github.com/shadow3x3x3/go-mock-exec-command-example
// which is based on https://github.com/golang/go/blob/master/src/os/exec/exec_test.go
func SetupTestCmd(fakeExecute func(command string, args ...string) *exec.Cmd) {
	execute = fakeExecute
}

// TearDownTestCmd recovers the execute command function
func TearDownTestCmd() {
	execute = exec.Command
}
