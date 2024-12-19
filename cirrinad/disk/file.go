//go:generate go run go.uber.org/mock/mockgen -destination=file_mocks.go -package=disk . FileInfoFetcher,LocalFileSystem

package disk

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type FileInfoFetcher interface {
	CheckExists(name string) (bool, error)
	Add(name string, size uint64) error
	FetchFileSize(name string) (uint64, error)
	FetchFileUsage(name string) (uint64, error)
	ApplyFileSize(volumeName string, volSize uint64) error
	FetchAll() ([]string, error)
}

type FileInfoCmds struct{}

type FileInfoService struct {
	FileInfoImpl FileInfoFetcher
}

type osFS struct{}

type LocalFileSystem interface {
	Open(name string) (MyFile, error)
	Stat(name string) (os.FileInfo, error)
}

type MyFile interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	Stat() (os.FileInfo, error)
}

var myFS LocalFileSystem = osFS{}

var myStat = syscall.Stat

var checkExistsFunc = FileInfoCmds.CheckExists

var fetchFileSizeFunc = FileInfoCmds.FetchFileSize

var currentUserFunc = user.Current

var utilOSReadDirFunc = util.OSReadDir

func NewFileInfoService(impl FileInfoFetcher) FileInfoService {
	if impl == nil {
		impl = &FileInfoCmds{}
	}

	d := FileInfoService{
		FileInfoImpl: impl,
	}

	return d
}

func (osFS) Open(name string) (MyFile, error) { return os.Open(name) } //nolint:wrapcheck

func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) } //nolint:wrapcheck

func (n FileInfoService) GetSize(name string) (uint64, error) {
	fileSize, err := n.FileInfoImpl.FetchFileSize(name)
	if err != nil {
		return 0, fmt.Errorf("error getting file size: %w", err)
	}

	return fileSize, nil
}

func (n FileInfoService) GetUsage(name string) (uint64, error) {
	fileUsage, err := n.FileInfoImpl.FetchFileUsage(name)
	if err != nil {
		return 0, fmt.Errorf("error getting file usage: %w", err)
	}

	return fileUsage, nil
}

func (n FileInfoService) SetSize(name string, newSize uint64) error {
	err := n.FileInfoImpl.ApplyFileSize(name, newSize)
	if err != nil {
		return fmt.Errorf("failed setting size: %w", err)
	}

	return nil
}

func (n FileInfoService) Exists(name string) (bool, error) {
	exists, err := n.FileInfoImpl.CheckExists(name)
	if err != nil {
		return true, fmt.Errorf("error checking file exists: %w", err)
	}

	return exists, nil
}

func (n FileInfoService) Create(name string, size uint64) error {
	err := n.FileInfoImpl.Add(name, size)

	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	return nil
}

func (n FileInfoService) GetAll() ([]string, error) {
	r, err := n.FileInfoImpl.FetchAll()
	if err != nil {
		return nil, fmt.Errorf("error fetching all: %w", err)
	}

	return r, nil
}

func (f FileInfoCmds) FetchFileSize(name string) (uint64, error) {
	diskFileStat, err := myFS.Stat(name)
	if err != nil {
		slog.Error("FetchFileSize error getting disk size", "err", err)

		return 0, fmt.Errorf("error stating disk path: %w", err)
	}

	// maybe we should check for nil interface, it can happen, happened during testing
	// if diskFileStat == nil {
	// 	 slog.Error("diskFileStat is nil")
	//
	//   return 0, errors.New("failed to stat")
	// }

	return uint64(diskFileStat.Size()), nil
}

func (f FileInfoCmds) FetchFileUsage(name string) (uint64, error) {
	var stat syscall.Stat_t

	var blockSize int64 = 512

	err := myStat(name, &stat)
	if err != nil {
		slog.Error("FetchFileUsage unable to stat diskPath", "diskPath", name, "err", err)

		return 0, fmt.Errorf("error stating disk file: %w", err)
	}

	return uint64(stat.Blocks * blockSize), nil
}

func (f FileInfoCmds) ApplyFileSize(name string, newSize uint64) error {
	exists, err := checkExistsFunc(f, name)
	if err != nil {
		return fmt.Errorf("failed checking disk: %w", err)
	}

	if !exists {
		return errDiskNotFound
	}

	curSize, err := fetchFileSizeFunc(f, name)
	if err != nil {
		return fmt.Errorf("failed checking disk: %w", err)
	}

	if newSize < curSize {
		return errDiskShrinkage
	}

	if curSize == newSize {
		return nil // nothing to do
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/truncate", "-s", strconv.FormatUint(newSize, 10), name},
	)
	if err != nil {
		slog.Error("failed to resize disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	return nil
}

func (f FileInfoCmds) CheckExists(name string) (bool, error) {
	// for files, just check the name
	diskPathExists, err := PathExistsFunc(name)
	if err != nil {
		slog.Error("error checking if disk exists", "name", name, "err", err)

		// assume disks exists if there's an error checking it, to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if diskPathExists {
		slog.Debug("disk exists", "name", name)

		return true, nil
	}

	return false, nil
}

func (f FileInfoCmds) Add(name string, size uint64) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/truncate", "-s", strconv.FormatUint(size, 10), name},
	)
	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	myUser, err := currentUserFunc()
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/chown", myUser.Username, name},
	)
	if err != nil {
		slog.Error("failed to fix ownership of disk file",
			"name", name,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed to fix ownership of disk file %s: %w", name, err)
	}

	return nil
}

func (f FileInfoCmds) FetchAll() ([]string, error) {
	fileNames, err := utilOSReadDirFunc(config.Config.Disk.VM.Path.Image)
	if err != nil {
		return nil, fmt.Errorf("error listing file volumes: %w", err)
	}

	return fileNames, nil
}

func (n FileInfoService) RemoveBacking(targetDisk *Disk) error {
	filePath := targetDisk.GetPath()

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error removing backing: %w", err)
	}

	return nil
}

func checkLeftoversFile() {
	diskFiles, err := os.ReadDir(config.Config.Disk.VM.Path.Image)
	if err != nil {
		slog.Error("failed checking disks", "err", err)

		return
	}

	for _, aDiskFile := range diskFiles {
		if !strings.HasSuffix(aDiskFile.Name(), ".img") {
			continue
		}

		diskName := strings.TrimSuffix(aDiskFile.Name(), ".img")
		if util.ValidDiskName(diskName) {
			_, err = GetByName(diskName)
			if err != nil {
				slog.Warn("possible left over disk (file)", "disk.Name", diskName, "file.Name", aDiskFile.Name())
			}
		}
	}
}
