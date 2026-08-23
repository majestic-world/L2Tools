package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const PROCESS_ALL_ACCESS = 0x1F0FFF

var (
	serverProcessID              uint32
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole            = kernel32.NewProc("AttachConsole")
	procSetConsoleCtrlHandler    = kernel32.NewProc("SetConsoleCtrlHandler")
	procGenerateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
	procFreeConsole              = kernel32.NewProc("FreeConsole")
	procSetConsoleTitleW         = kernel32.NewProc("SetConsoleTitleW")
)

type ServerConfig struct {
	ServerPath     string
	ServerCopyPath string
	JavaPath       string
	JavaArgs       string
	OutputJarPaths []string
	ExcludeJars    []string
}

func (c *ServerConfig) ResolvedCopyPath() string {
	if c.ServerCopyPath == "" {
		return c.ServerPath
	}
	return c.ServerCopyPath
}

func (c *ServerConfig) isExcludedJar(fileName string) bool {
	for _, ex := range c.ExcludeJars {
		if strings.EqualFold(ex, fileName) {
			return true
		}
	}
	return false
}

func loadConfig() *ServerConfig {
	exePath, _ := os.Executable()
	configPath := filepath.Join(filepath.Dir(exePath), "GameServer.properties")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		setColor("red")
		fmt.Println("Configuration file not found: GameServer.properties")
		fmt.Println("Press any key to exit...")
		resetColor()
		waitForKey()
		os.Exit(1)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		setColor("red")
		fmt.Printf("Error loading configuration: %v\n", err)
		resetColor()
		fmt.Println("\nPress any key to exit...")
		waitForKey()
		os.Exit(1)
	}

	config := &ServerConfig{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if idx := strings.Index(val, "#"); idx != -1 {
				val = strings.TrimSpace(val[:idx])
			}
			val = strings.Trim(val, "\"")

			switch key {
			case "ServerPath":
				config.ServerPath = val
			case "JavaPath":
				config.JavaPath = val
			case "JavaArgs":
				config.JavaArgs = val
			case "ServerCopyPath":
				config.ServerCopyPath = val
			case "OutputJarPath":
				paths := strings.Split(val, ";")
				for _, p := range paths {
					p = strings.TrimSpace(p)
					if p != "" {
						config.OutputJarPaths = append(config.OutputJarPaths, p)
					}
				}
			case "excludeJar":
				jars := strings.Split(val, ";")
				for _, j := range jars {
					j = strings.TrimSpace(j)
					if j != "" {
						config.ExcludeJars = append(config.ExcludeJars, j)
					}
				}
			}
		}
	}

	if config.ServerPath == "" || config.JavaArgs == "" || len(config.OutputJarPaths) == 0 {
		setColor("red")
		fmt.Println("Error: Configuration file is incomplete!")
		fmt.Println("Please check all required properties are set.")
		resetColor()
		fmt.Println("\nPress any key to exit...")
		waitForKey()
		os.Exit(1)
	}

	config.ServerPath = filepath.Clean(config.ServerPath)
	config.JavaPath = filepath.Clean(config.JavaPath)

	if info, err := os.Stat(config.ServerPath); err != nil || !info.IsDir() {
		setColor("red")
		fmt.Println("Error: ServerPath is not a valid directory:")
		fmt.Println(config.ServerPath)
		fmt.Println("\nThe JVM's working directory must exist on disk. Check GameServer.properties.")
		resetColor()
		fmt.Println("\nPress any key to exit...")
		waitForKey()
		os.Exit(1)
	}

	if info, err := os.Stat(config.JavaPath); err != nil || !info.IsDir() {
		setColor("red")
		fmt.Println("Error: JavaPath is not a valid directory:")
		fmt.Println(config.JavaPath)
		resetColor()
		fmt.Println("\nPress any key to exit...")
		waitForKey()
		os.Exit(1)
	}

	return config
}

func consoleCtrlHandler(ctrlType uint32) uintptr {
	switch ctrlType {
	case 0: // CTRL_C_EVENT: restart the server
		fmt.Println("\n\nRestart requested. Stopping server safely...")
		pid := atomic.LoadUint32(&serverProcessID)
		if pid != 0 {
			stopServerByPid(pid)
		}
		time.Sleep(2 * time.Second)
		return 1
	case 2: // CTRL_CLOSE_EVENT: console closing, stop and let the process exit
		fmt.Println("\n\nShutdown signal received. Stopping server safely...")
		pid := atomic.LoadUint32(&serverProcessID)
		if pid != 0 {
			stopServerByPid(pid)
		}
		time.Sleep(2 * time.Second)
		return 1
	}
	return 0
}

func setColor(color string) {
	switch color {
	case "red":
		fmt.Print("\033[31m")
	case "green":
		fmt.Print("\033[32m")
	case "yellow":
		fmt.Print("\033[33m")
	case "cyan":
		fmt.Print("\033[36m")
	}
}

func resetColor() {
	fmt.Print("\033[0m")
}

func enableVT() {
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	syscall.GetConsoleMode(h, &mode)
	mode |= 0x0004
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode").Call(uintptr(h), uintptr(mode))
}

func drawHeader() {
	setColor("cyan")
	fmt.Println("╔═══════════════════════════════════════╗")
	fmt.Println("║       GameServer Manager By Mk        ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	resetColor()
	fmt.Println()
}

func readChar() byte {
	h, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return 0
	}
	var mode uint32
	syscall.GetConsoleMode(h, &mode)
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode").Call(uintptr(h), uintptr(mode&^uint32(0x0002|0x0004)))

	buf := make([]byte, 1)
	os.Stdin.Read(buf)

	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode").Call(uintptr(h), uintptr(mode))
	return buf[0]
}

func waitForKey() {
	readChar()
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func startServer(config *ServerConfig) {
	clearScreen()
	drawHeader()

	updateJars(config)

	setColor("cyan")
	fmt.Println("Starting server... (Press Ctrl+C to restart safely)\n")
	resetColor()

	cmdPath := "java"
	if config.JavaPath != "" {
		cmdPath = filepath.Join(config.JavaPath, "java.exe")
		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			cmdPath = filepath.Join(config.JavaPath, "java")
		}
	}

	cmd := exec.Command(cmdPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `"` + cmdPath + `" ` + config.JavaArgs,
	}
	cmd.Dir = config.ServerPath

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		setColor("red")
		fmt.Printf("[ERROR] Failed to start process: %v\n", err)
		resetColor()
		return
	}

	pid := uint32(cmd.Process.Pid)
	atomic.StoreUint32(&serverProcessID, pid)

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			data := scanner.Text()
			if data == "" {
				continue
			}
			if strings.Contains(data, "ERROR") || strings.Contains(data, "Exception") {
				setColor("red")
			} else if strings.Contains(data, "WARNING") || strings.Contains(data, "WARN") {
				setColor("yellow")
			} else if strings.Contains(data, "Shutdown") || strings.Contains(data, "Saving") {
				setColor("cyan")
			}
			fmt.Println(data)
			resetColor()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			data := scanner.Text()
			if data != "" {
				setColor("red")
				fmt.Printf("[ERROR] %s\n", data)
				resetColor()
			}
		}
	}()

	err = cmd.Wait()
	atomic.StoreUint32(&serverProcessID, 0)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	} else if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	switch exitCode {
	case 2:
		setColor("yellow")
		fmt.Println("\n========================================")
		fmt.Println("Server Restarting...")
		fmt.Println("========================================")
		resetColor()
		time.Sleep(2 * time.Second)
	case 1:
		setColor("red")
		fmt.Println("\n========================================")
		fmt.Println("Server Terminated with Error!")
		fmt.Println("========================================")
		resetColor()
		fmt.Println("\nPress any key to continue...")
		waitForKey()
	}
}

func stopServerByPid(pid uint32) {
	if pid == 0 {
		fmt.Println("Server is not running.")
		return
	}

	setColor("yellow")
	fmt.Println("\n========================================")
	fmt.Println("Initiating Safe Shutdown...")
	fmt.Println("========================================")
	resetColor()

	r1, _, _ := procAttachConsole.Call(uintptr(pid))
	if r1 != 0 {
		procSetConsoleCtrlHandler.Call(0, 1)
		procGenerateConsoleCtrlEvent.Call(0, 0)
		time.Sleep(500 * time.Millisecond)
		procFreeConsole.Call()
		procSetConsoleCtrlHandler.Call(syscall.NewCallback(consoleCtrlHandler), 1)
	}

	h, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, false, pid)
	if err == nil {
		event, err := syscall.WaitForSingleObject(h, 30000)
		if err == nil && event == syscall.WAIT_TIMEOUT {
			setColor("red")
			fmt.Println("\nServer did not stop in time. Forcing shutdown...")
			resetColor()
			syscall.TerminateProcess(h, 1)
		} else {
			setColor("green")
			fmt.Println("\nServer stopped gracefully")
			resetColor()
		}
		syscall.CloseHandle(h)
	}
}

func updateJars(config *ServerConfig) {
	totalUpdated := 0

	for _, outputPath := range config.OutputJarPaths {
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			setColor("red")
			fmt.Printf("Output not found: %s\n", outputPath)
			resetColor()
			continue
		}

		entries, err := os.ReadDir(outputPath)
		if err != nil {
			continue
		}

		var jarFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jar") && !config.isExcludedJar(entry.Name()) {
				jarFiles = append(jarFiles, filepath.Join(outputPath, entry.Name()))
			}
		}

		if len(jarFiles) == 0 {
			continue
		}

		setColor("cyan")
		fmt.Printf("[%s]\n\n", outputPath)
		resetColor()

		for _, jarFile := range jarFiles {
			fileName := filepath.Base(jarFile)
			destFile := filepath.Join(config.ResolvedCopyPath(), fileName)

			if copyFile(jarFile, destFile) {
				fmt.Printf("%s (Ok)\n", fileName)
				totalUpdated++
			}
		}
	}

	if totalUpdated > 0 {
		fmt.Println()
		setColor("cyan")
		fmt.Printf("%d JAR(s) updated successfully.\n", totalUpdated)
		resetColor()
	}
}

func copyFile(src, dst string) bool {
	input, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	err = os.WriteFile(dst, input, 0644)
	return err == nil
}

func main() {
	enableVT()

	ptr, _ := syscall.UTF16PtrFromString("GameServer Manager By Mk")
	procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(ptr)))

	config := loadConfig()

	cb := syscall.NewCallback(consoleCtrlHandler)
	procSetConsoleCtrlHandler.Call(cb, 1)

	for {
		startServer(config)
	}
}
