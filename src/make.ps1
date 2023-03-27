Write-Output "Building Cultured Downloader CLI for Windows, Linux, and macOS..."

# Make the bin directory quietly if it doesn't exist
if (!(Test-Path -Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

Remove-Item -Path "bin/hash.txt" -Force -ErrorAction SilentlyContinue
"SHA256 Hashes`r`n" | Out-File -FilePath "bin/hash.txt"

function GetHash($path, $os, $arch) {
    $hash = Get-FileHash -Algorithm SHA256 $path | Select-Object -ExpandProperty Hash

    $bits = "64-bit"
    if ($arch -eq "386") {
        $bits = "32-bit"
    }

    $filename = Split-Path -Path $path -Leaf 
    $osTitle = $os.Substring(0,1).ToUpper() + $os.Substring(1)
    $hashMsg = "$filename ($os-$arch/$osTitle $bits):`r`n- $hash`r`n"

    # write to bin/hash.txt
    $hashMsg | Out-File -FilePath "bin/hash.txt" -Append
}

# github.com/josephspurrier/goversioninfo/cmd/goversioninfo
$verInfoName = "versioninfo.syso"
$verInfoRc = "versioninfo.rc"
windres -i $verInfoRc -O coff -o $verInfoName

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$binaryPath = "bin/cultured-downloader-cli.exe"
go build -o $binaryPath
GetHash $binaryPath "windows" "amd64"
Remove-Item -Path $verInfoName -Force -ErrorAction SilentlyContinue

$env:GOOS = "linux"
$binaryPath = "bin/cultured-downloader-cli-linux"
go build -o $binaryPath
GetHash $binaryPath "linux" "amd64"

$env:GOOS = "darwin"
$binaryPath = "bin/cultured-downloader-cli-darwin"
go build -o $binaryPath
GetHash $binaryPath "darwin" "amd64"

# reset the environment variables
$env:GOOS = "windows"
Write-Output "Finished building Cultured Downloader CLI for Windows, Linux, and macOS."
