package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	TH32CS_SNAPPROCESS                = 0x00000002
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	PROCESS_TERMINATE                 = 0x0001
	INVALID_HANDLE_VALUE              = ^uintptr(0)
)

// Layout matches Windows PROCESSENTRY32W on both 32/64-bit (Go adds implicit padding before DefaultHeapID).
type PROCESSENTRY32W struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTitleW           = kernel32.NewProc("SetConsoleTitleW")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procCreateToolhelp32Snapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW            = kernel32.NewProc("Process32FirstW")
	procProcess32NextW             = kernel32.NewProc("Process32NextW")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

func enableVT() {
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	syscall.GetConsoleMode(h, &mode)
	procSetConsoleMode.Call(uintptr(h), uintptr(mode|0x0004))
}

func setConsoleTitle(title string) {
	ptr, _ := syscall.UTF16PtrFromString(title)
	procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(ptr)))
}

func red(s string) string    { return "\033[31m" + s + "\033[0m" }
func yellow(s string) string { return "\033[33m" + s + "\033[0m" }

func pause() {
	fmt.Print(" Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func loadConfig() (map[string]string, error) {
	exePath, _ := os.Executable()
	configPath := filepath.Join(filepath.Dir(exePath), "config.properties")

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.IndexByte(line, '='); idx != -1 {
			config[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
		}
	}
	return config, scanner.Err()
}

func getProcessExePath(pid uint32) (string, error) {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)

	buf := make([]uint16, 260)
	size := uint32(260)
	r, _, e := procQueryFullProcessImageNameW.Call(
		uintptr(handle), 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return "", e
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

func closeL2Process(clientDir string) {
	fmt.Println(" Checking L2.exe process...")

	absClientDir, err := filepath.Abs(clientDir)
	if err != nil {
		absClientDir = clientDir
	}
	absClientDir = strings.ToLower(filepath.Clean(absClientDir))

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == INVALID_HANDLE_VALUE {
		return
	}
	defer syscall.CloseHandle(syscall.Handle(snapshot))

	var entry PROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	r, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for r != 0 {
		name := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		if name == "l2.exe" || name == "l2" {
			if exePath, err := getProcessExePath(entry.ProcessID); err == nil {
				procDir := strings.ToLower(filepath.Clean(filepath.Dir(exePath)))
				if procDir == absClientDir {
					if h, err := syscall.OpenProcess(PROCESS_TERMINATE, false, entry.ProcessID); err == nil {
						syscall.TerminateProcess(h, 0)
						syscall.CloseHandle(h)
					}
				}
			}
		}
		r, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
}

func saveCompilationLog(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	file, err := os.Create(filepath.Join(filepath.Dir(exePath), "log.txt"))
	if err != nil {
		return
	}
	defer file.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintln(file, "========================================")
	fmt.Fprintf(file, "Compilation Log - %s\n", ts)
	fmt.Fprintln(file, "========================================\n")
	fmt.Fprintln(file, "=== WARNINGS AND ERRORS ===\n")
	for _, w := range warnings {
		fmt.Fprintln(file, w)
	}
}

func runStrip(interfaceDir, commandlet string) {
	cmd := exec.Command("cmd", "/c", "UCC.exe "+commandlet+" Interface.u -nobind")
	cmd.Dir = interfaceDir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf(" %s\n", yellow("Warning: "+commandlet+" failed"))
		for _, line := range strings.Split(stdout.String(), "\n") {
			if line = strings.TrimRight(line, "\r"); line != "" {
				fmt.Printf("   %s\n", yellow(line))
			}
		}
		for _, line := range strings.Split(stderr.String(), "\n") {
			if line = strings.TrimRight(line, "\r"); line != "" {
				fmt.Printf("   %s\n", red(line))
			}
		}
	}
}

func compileInterface(interfaceDir string, deleteFiles []string, useStrip bool) error {
	fmt.Println(" Compiling interface...")

	for _, f := range deleteFiles {
		path := filepath.Join(interfaceDir, f)
		if _, err := os.Stat(path); err == nil {
			os.Remove(path)
		}
	}

	uccPath := filepath.Join(interfaceDir, "UCC.exe")
	if _, err := os.Stat(uccPath); os.IsNotExist(err) {
		return fmt.Errorf("UCC.exe not found: %s", uccPath)
	}

	cmd := exec.Command(uccPath, "make", "-nobind")
	cmd.Dir = interfaceDir

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var (
		mu       sync.Mutex
		warnings []string
		wg       sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf(" %s\n", red(line))
			mu.Lock()
			warnings = append(warnings, "[ERROR] "+line)
			mu.Unlock()
		}
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)
		isError := strings.Contains(lower, "error,") ||
			strings.Contains(lower, "compile aborted") ||
			strings.Contains(lower, "failure -")
		isWarning := strings.Contains(lower, "warning")

		switch {
		case isError:
			fmt.Printf(" %s\n", red(line))
			mu.Lock()
			warnings = append(warnings, line)
			mu.Unlock()
		case isWarning:
			fmt.Printf(" %s\n", yellow(line))
			mu.Lock()
			warnings = append(warnings, line)
			mu.Unlock()
		default:
			fmt.Printf(" %s\n", line)
		}
	}

	cmdErr := cmd.Wait()
	wg.Wait()

	saveCompilationLog(warnings)

	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			return fmt.Errorf("compilation failed with exit code: %d", exitErr.ExitCode())
		}
		return fmt.Errorf("compilation failed: %v", cmdErr)
	}

	if useStrip {
		fmt.Println(" Stripping interface...")
		runStrip(interfaceDir, "editor.stripsource")
		runStrip(interfaceDir, "editor.stripsourcecommandlet")
	}

	fmt.Println(" Compilation completed")
	return nil
}

func copyCompiledFile(interfaceDir, clientDir string) error {
	fmt.Println(" Copying compiled file...")
	src := filepath.Join(interfaceDir, "interface.u")
	dst := filepath.Join(clientDir, "interface.u")

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("compiled file not found: %s", src)
	}
	if _, err := os.Stat(clientDir); os.IsNotExist(err) {
		return fmt.Errorf("destination directory not found: %s", clientDir)
	}

	if info, err := os.Stat(dst); err == nil {
		if info.Mode().Perm()&0200 == 0 {
			os.Chmod(dst, info.Mode().Perm()|0200)
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	if n >= 1_048_576 {
		fmt.Printf(" File copied (%.2f MB)\n", float64(n)/1_048_576)
	} else {
		fmt.Printf(" File copied (%.2f KB)\n", float64(n)/1024)
	}
	return nil
}

func startL2(clientDir string) error {
	fmt.Println(" Starting L2.exe...")
	l2Path := filepath.Join(clientDir, "l2.exe")
	if _, err := os.Stat(l2Path); os.IsNotExist(err) {
		return fmt.Errorf("L2.exe not found: %s", l2Path)
	}
	cmd := exec.Command(l2Path)
	cmd.Dir = clientDir
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Println(" L2.exe started")
	return nil
}

func main() {
	enableVT()
	setConsoleTitle("Interface Build Assistant")

	start := time.Now()

	fmt.Println(" =================================")
	fmt.Println(" Interface Builder Launcher By Mk")
	fmt.Println(" =================================")

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, " Error loading config.properties: %v\n", err)
		pause()
		return
	}

	interfaceDir, ok := config["InterfaceDir"]
	if !ok {
		fmt.Fprintln(os.Stderr, " InterfaceDir not configured")
		pause()
		return
	}

	clientDir, ok := config["ClientDir"]
	if !ok {
		fmt.Fprintln(os.Stderr, " ClientDir not configured")
		pause()
		return
	}

	var deleteFiles []string
	if raw, ok := config["DeleteFiles"]; ok {
		for _, f := range strings.Split(raw, ",") {
			if f = strings.TrimSpace(f); f != "" {
				deleteFiles = append(deleteFiles, f)
			}
		}
	}

	useStrip := strings.ToLower(config["UseStrip"]) == "true"

	closeL2Process(clientDir)

	if err := compileInterface(interfaceDir, deleteFiles, useStrip); err != nil {
		fmt.Fprintf(os.Stderr, "\n Error: %v\n", err)
		pause()
		return
	}

	if err := copyCompiledFile(interfaceDir, clientDir); err != nil {
		fmt.Fprintf(os.Stderr, "\n Error: %v\n", err)
		pause()
		return
	}

	if err := startL2(clientDir); err != nil {
		fmt.Fprintf(os.Stderr, "\n Error: %v\n", err)
		pause()
		return
	}

	fmt.Println(" ================================")
	fmt.Println(" Process completed successfully!")
	fmt.Printf(" Compilation time: %.2fs\n", time.Since(start).Seconds())
	fmt.Println(" ================================")
	fmt.Println(" Auto-closing in 5 seconds...")
	time.Sleep(5 * time.Second)
}
