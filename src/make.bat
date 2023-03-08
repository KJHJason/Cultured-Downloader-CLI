@REM go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
windres -i versioninfo.rc -O coff -o versioninfo.syso

@REM 64-bit windows
set GOARCH=amd64
go build -o bin/windows/cultured-downloader-cli.exe

@REM 32-bit windows
set GOARCH=386
go build -o bin/windows/cultured-downloader-cli-32.exe

@REM 64-bit linux
set GOARCH=amd64
set GOOS=linux
go build -o bin/linux/cultured-downloader-cli

@REM 32-bit linux
set GOARCH=386
go build -o bin/linux/cultured-downloader-cli-32

@REM 64-bit mac
set GOARCH=amd64
set GOOS=darwin
go build -o bin/macos/cultured-downloader-cli

@REM 32-bit mac
set GOARCH=386
go build -o bin/macos/cultured-downloader-cli-32