# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Conventions

- All code and application-facing strings must be written in English.
- Never add comments to generated code.
- All chat responses must be in Brazilian Portuguese (pt-BR).
- Never include `Co-Authored-By` trailers in commit messages.

## Project Overview

**Majestic World L2 Dev Tools** — a Windows-only developer toolkit for managing Lineage 2 game server operations and interface compilation. Two executables are produced:

- **`GameServer.exe`** — Menu-driven CLI to start/stop/update a Lineage 2 game server (JAR hot-swap, graceful shutdown, auto-restart).
- **`Builder.exe`** — Automated pipeline: close L2 client → compile UnrealScript via UCC.exe → deploy `.u` file → relaunch client.

## Build Commands

**Prerequisites:**
- Go 1.24+
- GNU Make (`winget install GnuWin32.Make`)
- `goversioninfo` tool (`go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest`)

```bash
make build        # Compile both executables into build/
```

The Makefile runs `goversioninfo` in each `cmd/` subdirectory to generate `.syso` resource files (icons, version info, manifest), then compiles with `-ldflags="-s -w"`. Outputs: `build/GameServer.exe`, `build/Builder.exe`.

No test suite exists in this project.

## Architecture

```
cmd/
  GameServer/         # Server launcher
    main.go           # ~437 lines; all logic in one file
    versioninfo.json  # Windows resource metadata
  InterfaceBuilder/   # UCC build pipeline
    main.go           # ~422 lines; all logic in one file
    InterfaceBuilder.manifest  # Requires admin privileges
assets/               # .ico files embedded via goversioninfo
```

Both tools are **single-file, stdlib-only Go programs** — no external Go dependencies (`go.mod` has no `require` block).

### GameServer runtime flow
1. Reads `GameServer.properties` from the working directory.
2. Presents a 5-item menu (Start / Start+Update / UpdateJars / Stop / Exit).
3. Spawns the JVM via `os/exec`, attaches to its console for graceful `CTRL_C` shutdown.
4. On "UpdateJars": copies JARs from `OutputJarPath` (semicolon-delimited) to `ServerCopyPath`.
5. Auto-restarts on exit code 2; reports error on exit code 1.

### InterfaceBuilder runtime flow
1. Reads `config.properties` from the working directory.
2. Terminates any `L2.exe` in the configured `ClientDir` (Windows `CreateToolhelp32Snapshot` enumeration).
3. Runs `UCC.exe` compiler; parses stdout for errors/warnings.
4. Optionally strips source files (`UseStrip=true`).
5. Copies `interface.u` to `ClientDir`, then launches `L2.exe`.

### Windows API usage
Both tools use `syscall` / `golang.org/x/sys`-style direct `kernel32.dll` calls:
- Console color output (`SetConsoleTextAttribute`)
- Process enumeration and termination
- `CTRL_C_EVENT` signaling for graceful JVM shutdown

All code is Windows-specific; no cross-platform abstraction layer.

## Configuration (runtime, not compile-time)

`GameServer.properties` — must sit next to the executable:
```
ServerPath=<server root>
ServerCopyPath=<relative JAR deploy path>
JavaPath=<JDK bin dir>
JavaArgs=<JVM flags and main class>
OutputJarPath=<path1>;<path2>
```

`config.properties` — must sit next to Builder.exe:
```
InterfaceDir=<UnrealScript source dir>
ClientDir=<L2 client dir>
DeleteFiles=<comma-separated temp files>
UseStrip=true|false
```