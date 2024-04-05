#!powershell
#
# powershell -ExecutionPolicy Bypass -File .\scripts\deps_windows.ps1
#
# optionall pass `-DryRun` to only report what would be installed


# This script will install the dependencies needed to build Ollama from source on windows
# The script will do its best to detect existing installs of the dependencies so its idempotent
# however it does not check versions and wont upgrade any components
#
# You can opt-out of certain dependencies with XXX 

param(
    [Parameter(Mandatory = $false)]
    [switch]$DryRun,
    [Parameter(Mandatory = $false)]
    [switch]$SkipMinGW,
    [Parameter(Mandatory = $false)]
    [switch]$SkipMSVC,
    [Parameter(Mandatory = $false)]
    [switch]$SkipGolang,
    [Parameter(Mandatory = $false)]
    [switch]$SkipPerl,
    [Parameter(Mandatory = $false)]
    [switch]$SkipROCm,
    [Parameter(Mandatory = $false)]
    [switch]$SkipWinSDK,
    [Parameter(Mandatory = $false)]
    [switch]$SkipGCESignPlugin,
    [Parameter(Mandatory = $false)]
    [switch]$SkipCUDA
)

$mingw_url="https://github.com/msys2/msys2-installer/releases/download/2024-01-13/msys2-x86_64-20240113.exe"
$msvc_url="https://c2rsetup.officeapps.live.com/c2r/downloadVS.aspx?sku=community&channel=Release&version=VS2019"
$golang_url="https://go.dev/dl/go1.22.2.windows-amd64.msi"
# alt https://go.dev/dl/go1.22.2.windows-amd64.zip
$strawberry_url="https://github.com/StrawberryPerl/Perl-Dist-Strawberry/releases/download/SP_53822_64bit/strawberry-perl-5.38.2.2-64bit.msi"
$rocm_url="https://download.amd.com/developer/eula/rocm-hub/AMD-Software-PRO-Edition-23.Q4-WinSvr2022-For-HIP.exe"
$winsdk_url="https://go.microsoft.com/fwlink/p/?LinkId=323507"
$gcesignplugin_url="https://github.com/GoogleCloudPlatform/kms-integrations/releases/download/cng-v1.0/kmscng-1.0-windows-amd64.zip"
$cuda_url="https://developer.download.nvidia.com/compute/cuda/11.3.1/local_installers/cuda_11.3.1_465.89_win10.exe"

$ProgressPreference = 'SilentlyContinue'
Set-PSDebug -Trace 1


# Detect if we're running in github actions, and augment $env:GITHUB_ENV as we go to ensure tools are found in the path
if ("${env:RUNNER_TEMP}") {
    $script:TMPDIR=${env:RUNNER_TEMP}
    $script:ENV_FILE=$env:GITHUB_PATH
} else {
    # TODO remove once tested
    if ($null -eq "${env:TMPDIR}") { exit(1) }
    $script:TMPDIR=${env:TEMP}
    $script:ENV_FILE="${script:TMPDIR}\ollama.env"
}
write-host "TMP: ${script:TMPDIR}"

function MinGW() {
    if ($SkipMinGW) {
        return
    }
    # TODO is this the best thing to check for?
    $d=(get-command -ea 'silentlycontinue' gcc).path
    if ($d -ne $null) {
        write-host "MinGW (gcc) already present"
        return
    }
    $gcc=$(Resolve-Path -ea 'silentlycontinue' 'c:\msys64\ucrt64\bin\gcc.exe')
    if ($gcc) {
        write-host "MinGW (gcc) already present"
        return
    }
    if ($DryRun) {
        write-host "Would be installing MinGW"
        write-host "Skip with -SkipMinGW"
        return
    }
    
    write-host "Downloading MinGW"
    Invoke-WebRequest -Uri "${mingw_url}" -OutFile "$script:TMPDIR\mingw.exe"
    write-host "Installing MinGW"
    Start-Process "$script:TMPDIR\mingw.exe" -ArgumentList @(
        '--al',
        '-t', 'c:\msys64',
        '--da',
        '-c',
        'install'
        ) -NoNewWindow -Wait
    write-host "Completed MinGW, now installing GCC"

    Start-Process "c:\msys64\usr\bin\pacman.exe" -ArgumentList @(
        '--noconfirm',
        '-S', 'mingw-w64-ucrt-x86_64-gcc'
        ) -NoNewWindow -Wait 
    write-host "Completed installing GCC"

    # TODO somehow get the PATH updated
    # c:\msys64\usr\bin;c:\msys64\ucrt64\bin
}

# Check for existence of the MSVC Community Edition
function MSVC() {
    if ($SkipMSVC) {
        return
    }

    $d=(get-command -ea 'silentlycontinue' cl).path
    if ($d -ne $null) {
        write-host "MSVC already present"
        return
    }
    if ($DryRun) {
        write-host "Would be installing MSVC"
        write-host "Skip with -SkipMSVC"
        return
    }

    write-host "Downloading MSVC Community Edition"
    Invoke-WebRequest -Uri "${msvc_url}" -OutFile "$script:TMPDIR\msvc.exe"
    write-host "Installing MSVC Community Edition"

    # TODO - Not actually installing/working yet, just silently does nothing?!.
    
    Start-Process "$script:TMPDIR\msvc.exe" -ArgumentList @(
        "--quiet",
        "--productId", "Microsoft.VisualStudio.Product.Community",
        "--productId", "Microsoft.VisualStudio.Product.BuildTools",
        "--all"
        ) -NoNewWindow -Wait
    write-host "Completed msvc"
}

function Golang() {
    if ($SkipGolang) {
        return
    }
    $d=(get-command -ea 'silentlycontinue' go).path
    if ($d -ne $null) {
        write-host "Go already present"
        return
    }
    if ($DryRun) {
        write-host "Would be installing Go"
        write-host "Skip with -SkipGolang"
        return
    }

    write-host "Downloading Go"
    Invoke-WebRequest -Uri "${golang_url}" -OutFile "$script:TMPDIR\golang.msi"
    write-host "Installing Go"

    # TODO - Not actually installing/working yet, just silently does nothing?!.

    Start-Process msiexec "/i $script:TMPDIR\golang.msi /norestart /qn" -Wait;
    write-host "Completed Go"
}


function ROCM() {
    if ($SkipROCm) {
        return
    }
    if ($env:HIP_PATH) {
        write-host "ROCm already present"
        return
    }
    $hip_clang=$(Resolve-Path -ea 'silentlycontinue' 'C:\Program Files\AMD\ROCm\*\bin\clang.exe')
    if ($hip_clang) {
        write-host "ROCm already present"
        return
    }
    if ($DryRun) {
        write-host "Would be installing ROCm"
        write-host "Skip with -SkipROCm"
        return
    }
    write-host "Downloading ROCm"
    Invoke-WebRequest -Uri "${rocm_url}" -OutFile "$script:TMPDIR\rocm.exe"
    write-host "Installing ROCm"
    Start-Process "$script:TMPDIR\rocm.exe" -ArgumentList '-install' -NoNewWindow -Wait -Verb RunAs
    write-host "Completed ROCm"

    # Note: perl from c:\msys64\usr\bin
}

function WinSDK() {
    if ($SkipWinSDK) {
        return
    }
    $signtool=$(Resolve-Path -ea 'silentlycontinue' 'C:\Program Files (x86)\Windows Kits\8.1\bin\x64\signtool.exe')
    if ($null -ne $signtool) {
        write-host "Required WinSDK version present"
        return
    }
    if ($DryRun) {
        write-host "Would be installing Win SDK"
        write-host "Skip with -SkipWinSDK"
        return
    }

    write-host "Downloading WinSDK"
    Invoke-WebRequest -Uri "${winsdk_url}" -OutFile "${script:TMPDIR}\sdksetup.exe"
    Start-Process "${script:TMPDIR}\sdksetup.exe" -ArgumentList @("/q") -NoNewWindow -Wait
    write-host "WinSDK installed"
}

function GCESignPlugin() {
    if ($SkipGCESignPlugin) {
        return
    }

    # TODO implement the detection algorithm

    if ($DryRun) {
        write-host "Would be installing GCE Signing Plugin"
        write-host "Skip with -SkipGCESignPlugin"
        return
    }

    write-host "Downloading GCE Signing Plugin"
    Invoke-WebRequest -Uri "${gcesignplugin_url}" -OutFile "${script:TMPDIR}\plugin.zip"
    Expand-Archive -Path "${script:TMPDIR}\plugin.zip" -DestinationPath ${script:TMPDIR}\plugin\
    write-host "Installing plugin"
    Start-Process "${env:RUNNER_TEMP}\plugin\*\kmscng.msi" -ArgumentList @("/quiet") -NoNewWindow -Wait
    write-host "plugin installed"
}

function CUDA() {
    if ($SkipCUDA) {
        return
    }
    $d=(get-command -ea 'silentlycontinue' nvcc).path
    if ($d -ne $null) {
        write-host "CUDA already present"
        return
    }
    
    # TODO should we check the standard location(s) too?

    if ($DryRun) {
        write-host "Would be installing CUDA"
        write-host "Skip with -SkipCUDA"
        return
    }
    write-host "Downloading CUDA"
    Invoke-WebRequest -Uri "${cuda_url}" -OutFile "$script:TMPDIR\cuda.exe"
    write-host "Installing CUDA"
    Start-Process "$script:TMPDIR\cuda.exe" -ArgumentList '-s' -NoNewWindow -Wait
    write-host "Completed CUDA"
    $cudaPath=((resolve-path "c:\Program Files\NVIDIA*\CUDA\v*\bin\nvcc.exe")[0].path | split-path | split-path)
    $cudaVer=($cudaPath | split-path -leaf ) -replace 'v(\d+).(\d+)', '$1_$2' 
    echo "$cudaPath\bin" >> "$script:ENV_FILE"
    echo "CUDA_PATH=$cudaPath" >> "$script:ENV_FILE"
    echo "CUDA_PATH_V${cudaVer}=$cudaPath" >> "$script:ENV_FILE"
    echo "CUDA_PATH_VX_Y=CUDA_PATH_V${cudaVer}" >> "$script:ENV_FILE"
}


try {
    MinGW
    MSVC
    Golang
    ROCM
    CUDA
    if ("${env:KEY_CONTAINER}") {
        Write-host "Code signing enabled, verifying dependencies"
        WinSDK
        GCESignPlugin
    } else {
        write-host "Code signing disabled - skipping signing dependencies"
    }
    
} catch {
    write-host "Dependency install Failed"
    write-host $_
}
