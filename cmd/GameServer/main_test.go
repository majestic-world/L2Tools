package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const synchronize = 0x00100000

func TestKillOnCloseJobStopsInheritedJavaProcess(t *testing.T) {
	if os.Getenv("GAMESERVER_JOB_HELPER") == "1" {
		job, err := newKillOnCloseJob()
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
		if err := job.assignCurrentProcess(); err != nil {
			t.Fatalf("assign manager to job: %v", err)
		}

		child := exec.Command("cmd.exe", "/C", "ping -n 60 127.0.0.1 > NUL")
		if err := child.Start(); err != nil {
			t.Fatalf("start child: %v", err)
		}
		fmt.Println(child.Process.Pid)
		return
	}

	helper := exec.Command(os.Args[0], "-test.run=^TestKillOnCloseJobStopsInheritedJavaProcess$")
	helper.Env = append(os.Environ(), "GAMESERVER_JOB_HELPER=1")
	helper.Stderr = os.Stderr
	stdout, err := helper.StdoutPipe()
	if err != nil {
		t.Fatalf("create helper stdout pipe: %v", err)
	}
	if err := helper.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatalf("read child PID from helper: %v", scanner.Err())
	}
	pid, err := strconv.ParseUint(scanner.Text(), 10, 32)
	if err != nil {
		t.Fatalf("parse child PID %q: %v", scanner.Text(), err)
	}
	if err := helper.Wait(); err != nil {
		t.Fatalf("helper exited unexpectedly: %v", err)
	}

	childHandle, err := syscall.OpenProcess(synchronize, false, uint32(pid))
	if err != nil {
		t.Fatalf("open inherited child process: %v", err)
	}
	defer syscall.CloseHandle(childHandle)

	status, err := syscall.WaitForSingleObject(childHandle, uint32(time.Second/time.Millisecond*5))
	if err != nil {
		t.Fatalf("wait for inherited child: %v", err)
	}
	if status != syscall.WAIT_OBJECT_0 {
		t.Fatalf("inherited child remained alive after manager exit; wait status = %d", status)
	}
}
