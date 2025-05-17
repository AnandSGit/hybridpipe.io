# Kafka Installation Script for Windows
#
# This script installs Apache Kafka on Windows.
# It downloads Kafka, extracts it, and configures it.
#
# Usage:
#   .\kafka_windows.ps1 [options]
#
# Options:
#   -Version <version>       Kafka version to install (default: 3.5.1)
#   -ScalaVersion <version>  Scala version to use (default: 2.13)
#   -InstallDir <path>       Installation directory (default: C:\kafka)
#   -ZooKeeperPort <port>    ZooKeeper port (default: 2181)
#   -KafkaPort <port>        Kafka port (default: 9092)
#   -StartServices           Start ZooKeeper and Kafka services after installation
#   -CreateService           Create Windows services for ZooKeeper and Kafka
#   -Verbose                 Show verbose output

param (
    [string]$Version = "3.5.1",
    [string]$ScalaVersion = "2.13",
    [string]$InstallDir = "C:\kafka",
    [int]$ZooKeeperPort = 2181,
    [int]$KafkaPort = 9092,
    [switch]$StartServices,
    [switch]$CreateService,
    [switch]$Verbose
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Enable verbose output if requested
if ($Verbose) {
    $VerbosePreference = "Continue"
}

# Function to log messages
function Log-Message {
    param (
        [string]$Message,
        [string]$Type = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Type] $Message"
}

# Function to check if a command exists
function Test-Command {
    param (
        [string]$Command
    )
    
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

# Function to download a file
function Download-File {
    param (
        [string]$Url,
        [string]$OutputFile
    )
    
    Log-Message "Downloading $Url to $OutputFile"
    
    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($Url, $OutputFile)
        Log-Message "Download completed successfully"
    }
    catch {
        Log-Message "Failed to download file: $_" "ERROR"
        exit 1
    }
}

# Function to extract a zip file
function Extract-ZipFile {
    param (
        [string]$ZipFile,
        [string]$OutputDir
    )
    
    Log-Message "Extracting $ZipFile to $OutputDir"
    
    try {
        Expand-Archive -Path $ZipFile -DestinationPath $OutputDir -Force
        Log-Message "Extraction completed successfully"
    }
    catch {
        Log-Message "Failed to extract file: $_" "ERROR"
        exit 1
    }
}

# Function to create a directory
function Create-Directory {
    param (
        [string]$Path
    )
    
    if (-not (Test-Path $Path)) {
        Log-Message "Creating directory $Path"
        New-Item -Path $Path -ItemType Directory -Force | Out-Null
    }
}

# Function to update a configuration file
function Update-ConfigFile {
    param (
        [string]$FilePath,
        [hashtable]$Properties
    )
    
    Log-Message "Updating configuration file $FilePath"
    
    $content = Get-Content $FilePath
    
    foreach ($key in $Properties.Keys) {
        $value = $Properties[$key]
        $pattern = "^#?\s*$key\s*=.*"
        $replacement = "$key=$value"
        $found = $false
        
        $content = $content | ForEach-Object {
            if ($_ -match $pattern) {
                $found = $true
                $replacement
            }
            else {
                $_
            }
        }
        
        if (-not $found) {
            $content += $replacement
        }
    }
    
    $content | Set-Content $FilePath
}

# Function to create a Windows service
function Create-WindowsService {
    param (
        [string]$ServiceName,
        [string]$DisplayName,
        [string]$Description,
        [string]$BinaryPath
    )
    
    Log-Message "Creating Windows service $ServiceName"
    
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    
    if ($service) {
        Log-Message "Service $ServiceName already exists" "WARNING"
        return
    }
    
    try {
        $nssm = "C:\ProgramData\chocolatey\bin\nssm.exe"
        
        if (-not (Test-Path $nssm)) {
            Log-Message "Installing NSSM (Non-Sucking Service Manager)"
            choco install nssm -y
        }
        
        & $nssm install $ServiceName $BinaryPath
        & $nssm set $ServiceName DisplayName $DisplayName
        & $nssm set $ServiceName Description $Description
        & $nssm set $ServiceName Start SERVICE_AUTO_START
        
        Log-Message "Service $ServiceName created successfully"
    }
    catch {
        Log-Message "Failed to create service: $_" "ERROR"
        exit 1
    }
}

# Check if running as administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Log-Message "This script must be run as Administrator" "ERROR"
    exit 1
}

# Check if Java is installed
if (-not (Test-Command "java")) {
    Log-Message "Java is not installed. Installing OpenJDK 17..."
    
    if (-not (Test-Command "choco")) {
        Log-Message "Chocolatey is not installed. Installing Chocolatey..."
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
    }
    
    choco install openjdk17 -y
    
    if (-not (Test-Command "java")) {
        Log-Message "Failed to install Java" "ERROR"
        exit 1
    }
}

# Create installation directory
Create-Directory $InstallDir

# Download Kafka
$kafkaVersion = "kafka_$ScalaVersion-$Version"
$kafkaUrl = "https://downloads.apache.org/kafka/$Version/$kafkaVersion.tgz"
$kafkaArchive = "$env:TEMP\$kafkaVersion.tgz"

Download-File $kafkaUrl $kafkaArchive

# Extract Kafka
$tempDir = "$env:TEMP\$kafkaVersion"
Create-Directory $tempDir

Log-Message "Extracting Kafka archive"
tar -xzf $kafkaArchive -C $env:TEMP

# Move Kafka to installation directory
Log-Message "Moving Kafka to installation directory"
Copy-Item -Path "$tempDir\*" -Destination $InstallDir -Recurse -Force

# Create data directories
$zooKeeperDataDir = "$InstallDir\data\zookeeper"
$kafkaDataDir = "$InstallDir\data\kafka"

Create-Directory $zooKeeperDataDir
Create-Directory $kafkaDataDir

# Update ZooKeeper configuration
$zooKeeperConfig = @{
    "dataDir" = $zooKeeperDataDir.Replace("\", "/")
    "clientPort" = $ZooKeeperPort
    "maxClientCnxns" = 0
}

Update-ConfigFile "$InstallDir\config\zookeeper.properties" $zooKeeperConfig

# Update Kafka configuration
$kafkaConfig = @{
    "broker.id" = 0
    "listeners" = "PLAINTEXT://localhost:$KafkaPort"
    "advertised.listeners" = "PLAINTEXT://localhost:$KafkaPort"
    "log.dirs" = $kafkaDataDir.Replace("\", "/")
    "zookeeper.connect" = "localhost:$ZooKeeperPort"
    "num.partitions" = 1
    "default.replication.factor" = 1
    "offsets.topic.replication.factor" = 1
    "transaction.state.log.replication.factor" = 1
    "transaction.state.log.min.isr" = 1
}

Update-ConfigFile "$InstallDir\config\server.properties" $kafkaConfig

# Create batch files for starting ZooKeeper and Kafka
$zooKeeperBat = @"
@echo off
set KAFKA_HOME=$InstallDir
cd %KAFKA_HOME%
bin\windows\zookeeper-server-start.bat config\zookeeper.properties
"@

$kafkaBat = @"
@echo off
set KAFKA_HOME=$InstallDir
cd %KAFKA_HOME%
bin\windows\kafka-server-start.bat config\server.properties
"@

$zooKeeperBat | Set-Content "$InstallDir\start-zookeeper.bat"
$kafkaBat | Set-Content "$InstallDir\start-kafka.bat"

# Create Windows services if requested
if ($CreateService) {
    if (-not (Test-Command "choco")) {
        Log-Message "Chocolatey is not installed. Installing Chocolatey..."
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
    }
    
    Create-WindowsService "ZooKeeper" "Apache ZooKeeper" "Apache ZooKeeper service for Kafka" "$InstallDir\start-zookeeper.bat"
    Create-WindowsService "Kafka" "Apache Kafka" "Apache Kafka message broker service" "$InstallDir\start-kafka.bat"
}

# Start services if requested
if ($StartServices) {
    if ($CreateService) {
        Log-Message "Starting ZooKeeper service"
        Start-Service -Name "ZooKeeper"
        
        Log-Message "Waiting for ZooKeeper to start"
        Start-Sleep -Seconds 10
        
        Log-Message "Starting Kafka service"
        Start-Service -Name "Kafka"
    }
    else {
        Log-Message "Starting ZooKeeper"
        Start-Process -FilePath "$InstallDir\start-zookeeper.bat" -WindowStyle Minimized
        
        Log-Message "Waiting for ZooKeeper to start"
        Start-Sleep -Seconds 10
        
        Log-Message "Starting Kafka"
        Start-Process -FilePath "$InstallDir\start-kafka.bat" -WindowStyle Minimized
    }
}

# Clean up temporary files
Remove-Item $kafkaArchive -Force
Remove-Item $tempDir -Recurse -Force

Log-Message "Kafka installation completed successfully"
Log-Message "ZooKeeper is configured to run on port $ZooKeeperPort"
Log-Message "Kafka is configured to run on port $KafkaPort"
Log-Message "To start ZooKeeper: $InstallDir\start-zookeeper.bat"
Log-Message "To start Kafka: $InstallDir\start-kafka.bat"
