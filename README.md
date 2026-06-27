# Majestic World L2 Dev Tools

## Requirements

Install `make` via winget:

```
winget install GnuWin32.Make
```

After installation, restart your terminal so the `make` command is available in PATH.

## Build

```
make build
```

Compiled binaries will be placed in the `build/` directory.

## GameServer.properties

```
ServerPath="C:\Users\Mk\Desktop\MyServer\gameserver"
ServerCopyPath="libs" 
JavaPath="C:\Users\Mk\Documents\jdk-25.0.2\bin\"
JavaArgs="-server -Dfile.encoding=UTF-8 -Xmx8G -cp config;./libs/* l2.gameserver.GameServer"
OutputJarPath="C:\workspace\java\Majestic-Pack\build\artifacts"
```