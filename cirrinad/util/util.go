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

	pathFile, err := os.Open(path)
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

// runCommandAndCaptureOutput does what it says on the tin. maybe I should make a version that takes a string to pass
// in on standard input. also maybe I should make this public here. perhaps pointer args would be better
func runCommandAndCaptureOutput(cmdName string, cmdArgs []string) ([]byte, error) {
	var outResult []byte

	var errResult []byte

	var errStdout, errStderr error

	cmd := exec.Command(cmdName, cmdArgs...)

	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, fmt.Errorf("error running command: %w", err)
	}

	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		return []byte{}, fmt.Errorf("error running command: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return []byte{}, fmt.Errorf("error running command: %w", err)
	}

	var runCmdWaitGroup sync.WaitGroup

	runCmdWaitGroup.Add(1)

	go func() {
		outResult, errStdout = captureReader(stdoutIn)

		runCmdWaitGroup.Done()
	}()

	errResult, errStderr = captureReader(stderrIn)

	runCmdWaitGroup.Wait()

	if errStdout != nil {
		return []byte{}, errStderr
	}

	if errStderr != nil {
		return []byte{}, errStderr
	}

	if len(errResult) > 0 {
		return []byte{}, errSTDERRNotEmpty
	}

	if err := cmd.Wait(); err != nil {
		return []byte{}, fmt.Errorf("error running command: %w", err)
	}

	return outResult, nil
}

func getNetstatJSONOutput() ([]byte, error) {
	return runCommandAndCaptureOutput("/usr/bin/netstat", []string{"-an", "--libxo", "json"})
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
	netstatJSON, err := getNetstatJSONOutput()
	if err != nil {
		return 0, err
	}

	uniqueLocalListenPorts, err := parseNetstatJSONOutput(netstatJSON)
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

func GetIntGroups(interfaceName string) ([]string, error) {
	var intGroups []string

	var err error

	cmd := exec.Command("/sbin/ifconfig", interfaceName)
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []string{}, fmt.Errorf("error running ifconfig: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return []string{}, fmt.Errorf("error running ifconfig: %w", err)
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
		return []string{}, fmt.Errorf("error parsing ifconfig output: %w", err)
	}

	return intGroups, nil
}

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

func ValidDiskName(name string) bool {
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

func CheckInRange(name string, myRT *unicode.RangeTable) bool {
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

		return 0, fmt.Errorf("error running sysctl: %w", err)
	}

	maxCPUStr := strings.TrimSpace(outBytes.String())

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
	hostCpus, err := GetHostMaxVMCpus()
	if err != nil {
		slog.Error("error getting number of host cpus", "err", err)

		return false
	}

	if numCpus > hostCpus {
		return false
	}

	return true
}
