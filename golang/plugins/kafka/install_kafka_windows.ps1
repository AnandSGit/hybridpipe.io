# ============================================================================
# Kafka Installation Script for Windows
# ============================================================================
# This script automates the installation and configuration of Apache Kafka
# and its dependencies (Java, ZooKeeper) on Windows systems.
#
# Author: HybridPipe.io
# Date: April 20, 2025
# Version: 1.0
# ============================================================================

param (
    [string]$KafkaVersion = "3.5.1",
    [string]$ScalaVersion = "2.13",
    [string]$ZooKeeperVersion = "3.8.1",
    [string]$JavaVersion = "17",
    [int]$BrokerId = 0,
    [string]$ZooKeeperHost = "localhost",
    [int]$ZooKeeperPort = 2181,
    [string]$KafkaPort = 9092,
    [string]$InstallDir = "$env:ProgramFiles\Kafka",
    [string]$DataDir = "$env:ProgramData\Kafka",
    [string]$LogDir = "$env:ProgramData\Kafka\logs",
    [switch]$InstallZooKeeper = $true,
    [switch]$InstallJava = $true,
    [switch]$StartServices = $true,
    [switch]$CreateServices = $true,
    [switch]$Verbose = $false,
    [switch]$Help = $false
)

# ============================================================================
# Function Definitions
# ============================================================================

function Write-Log {
    param (
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    
    if ($Level -eq "ERROR") {
        Write-Host $logMessage -ForegroundColor Red
    } elseif ($Level -eq "WARNING") {
        Write-Host $logMessage -ForegroundColor Yellow
    } elseif ($Level -eq "SUCCESS") {
        Write-Host $logMessage -ForegroundColor Green
    } else {
        if ($Verbose) {
            Write-Host $logMessage -ForegroundColor Cyan
        }
    }
    
    # Also append to log file
    $logFilePath = Join-Path $env:TEMP "kafka_install.log"
    Add-Content -Path $logFilePath -Value $logMessage
}

function Show-Help {
    Write-Host "Kafka Installation Script for Windows" -ForegroundColor Green
    Write-Host "====================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "This script installs and configures Apache Kafka and its dependencies on Windows."
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -KafkaVersion <version>     : Kafka version to install (default: 3.5.1)"
    Write-Host "  -ScalaVersion <version>     : Scala version for Kafka (default: 2.13)"
    Write-Host "  -ZooKeeperVersion <version> : ZooKeeper version to install (default: 3.8.1)"
    Write-Host "  -JavaVersion <version>      : Java version to install (default: 17)"
    Write-Host "  -BrokerId <id>              : Kafka broker ID (default: 0)"
    Write-Host "  -ZooKeeperHost <host>       : ZooKeeper host (default: localhost)"
    Write-Host "  -ZooKeeperPort <port>       : ZooKeeper port (default: 2181)"
    Write-Host "  -KafkaPort <port>           : Kafka port (default: 9092)"
    Write-Host "  -InstallDir <path>          : Installation directory (default: $env:ProgramFiles\Kafka)"
    Write-Host "  -DataDir <path>             : Data directory (default: $env:ProgramData\Kafka)"
    Write-Host "  -LogDir <path>              : Log directory (default: $env:ProgramData\Kafka\logs)"
    Write-Host "  -InstallZooKeeper           : Install ZooKeeper (default: true)"
    Write-Host "  -InstallJava                : Install Java (default: true)"
    Write-Host "  -StartServices              : Start services after installation (default: true)"
    Write-Host "  -CreateServices             : Create Windows services for Kafka and ZooKeeper (default: true)"
    Write-Host "  -Verbose                    : Enable verbose logging (default: false)"
    Write-Host "  -Help                       : Show this help message"
    Write-Host ""
    Write-Host "Example:"
    Write-Host "  .\install_kafka_windows.ps1 -KafkaVersion 3.5.1 -BrokerId 1 -Verbose"
    Write-Host ""
}

function Test-Administrator {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $isAdmin = $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    return $isAdmin
}

function Install-JavaWithChoco {
    param (
        [string]$Version
    )
    
    Write-Log "Installing Java $Version using Chocolatey..."
    
    # Check if Chocolatey is installed
    if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
        Write-Log "Chocolatey not found. Installing Chocolatey..." "INFO"
        try {
            Set-ExecutionPolicy Bypass -Scope Process -Force
            [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
            Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
            Write-Log "Chocolatey installed successfully" "SUCCESS"
        } catch {
            Write-Log "Failed to install Chocolatey: $_" "ERROR"
            return $false
        }
    }
    
    # Install Java
    try {
        choco install openjdk$Version -y
        Write-Log "Java $Version installed successfully" "SUCCESS"
        
        # Set JAVA_HOME environment variable
        $javaPath = "$env:ProgramFiles\OpenJDK\jdk-$Version"
        if (Test-Path $javaPath) {
            [System.Environment]::SetEnvironmentVariable("JAVA_HOME", $javaPath, [System.EnvironmentVariableTarget]::Machine)
            Write-Log "JAVA_HOME environment variable set to $javaPath" "SUCCESS"
            return $true
        } else {
            Write-Log "Java installation path not found at $javaPath" "WARNING"
            return $false
        }
    } catch {
        Write-Log "Failed to install Java: $_" "ERROR"
        return $false
    }
}

function Download-File {
    param (
        [string]$Url,
        [string]$OutputPath
    )
    
    Write-Log "Downloading $Url to $OutputPath..." "INFO"
    
    try {
        # Create directory if it doesn't exist
        $directory = Split-Path -Path $OutputPath -Parent
        if (-not (Test-Path $directory)) {
            New-Item -ItemType Directory -Path $directory -Force | Out-Null
        }
        
        # Download file
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($Url, $OutputPath)
        
        if (Test-Path $OutputPath) {
            Write-Log "Downloaded successfully to $OutputPath" "SUCCESS"
            return $true
        } else {
            Write-Log "Failed to download file to $OutputPath" "ERROR"
            return $false
        }
    } catch {
        Write-Log "Failed to download file: $_" "ERROR"
        return $false
    }
}

function Extract-Archive {
    param (
        [string]$ArchivePath,
        [string]$DestinationPath
    )
    
    Write-Log "Extracting $ArchivePath to $DestinationPath..." "INFO"
    
    try {
        # Create destination directory if it doesn't exist
        if (-not (Test-Path $DestinationPath)) {
            New-Item -ItemType Directory -Path $DestinationPath -Force | Out-Null
        }
        
        # Extract archive
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($ArchivePath, $DestinationPath)
        
        Write-Log "Extracted successfully to $DestinationPath" "SUCCESS"
        return $true
    } catch {
        Write-Log "Failed to extract archive: $_" "ERROR"
        return $false
    }
}

function Install-ZooKeeper {
    param (
        [string]$Version,
        [string]$InstallDir,
        [string]$DataDir
    )
    
    Write-Log "Installing ZooKeeper $Version..." "INFO"
    
    # Download ZooKeeper
    $zooKeeperUrl = "https://dlcdn.apache.org/zookeeper/zookeeper-$Version/apache-zookeeper-$Version-bin.zip"
    $zooKeeperArchive = Join-Path $env:TEMP "apache-zookeeper-$Version-bin.zip"
    
    if (-not (Download-File -Url $zooKeeperUrl -OutputPath $zooKeeperArchive)) {
        return $false
    }
    
    # Extract ZooKeeper
    $zooKeeperDir = Join-Path $InstallDir "apache-zookeeper-$Version-bin"
    if (-not (Extract-Archive -ArchivePath $zooKeeperArchive -DestinationPath $InstallDir)) {
        return $false
    }
    
    # Create ZooKeeper data directory
    $zooKeeperDataDir = Join-Path $DataDir "zookeeper"
    if (-not (Test-Path $zooKeeperDataDir)) {
        New-Item -ItemType Directory -Path $zooKeeperDataDir -Force | Out-Null
    }
    
    # Create ZooKeeper configuration
    $zooKeeperConfig = @"
tickTime=2000
dataDir=$($zooKeeperDataDir.Replace("\", "/"))
clientPort=$ZooKeeperPort
initLimit=5
syncLimit=2
"@
    
    $zooKeeperConfigPath = Join-Path $zooKeeperDir "conf\zoo.cfg"
    Set-Content -Path $zooKeeperConfigPath -Value $zooKeeperConfig
    
    Write-Log "ZooKeeper $Version installed successfully" "SUCCESS"
    return $true
}

function Install-Kafka {
    param (
        [string]$Version,
        [string]$ScalaVersion,
        [string]$InstallDir,
        [string]$DataDir,
        [string]$LogDir,
        [int]$BrokerId,
        [string]$ZooKeeperHost,
        [int]$ZooKeeperPort,
        [int]$KafkaPort
    )
    
    Write-Log "Installing Kafka $Version (Scala $ScalaVersion)..." "INFO"
    
    # Download Kafka
    $kafkaUrl = "https://dlcdn.apache.org/kafka/$Version/kafka_$ScalaVersion-$Version.tgz"
    $kafkaArchive = Join-Path $env:TEMP "kafka_$ScalaVersion-$Version.tgz"
    
    if (-not (Download-File -Url $kafkaUrl -OutputPath $kafkaArchive)) {
        return $false
    }
    
    # Extract Kafka (requires 7-Zip for .tgz files on Windows)
    $kafkaDir = Join-Path $InstallDir "kafka_$ScalaVersion-$Version"
    
    # Check if 7-Zip is installed
    $7zipPath = "$env:ProgramFiles\7-Zip\7z.exe"
    if (-not (Test-Path $7zipPath)) {
        Write-Log "7-Zip not found. Installing 7-Zip..." "INFO"
        try {
            choco install 7zip -y
            Write-Log "7-Zip installed successfully" "SUCCESS"
        } catch {
            Write-Log "Failed to install 7-Zip: $_" "ERROR"
            return $false
        }
    }
    
    # Create Kafka installation directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    # Extract Kafka using 7-Zip
    Write-Log "Extracting Kafka archive..." "INFO"
    $tempDir = Join-Path $env:TEMP "kafka_extract"
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    # Extract .tgz file (two steps with 7-Zip)
    & "$env:ProgramFiles\7-Zip\7z.exe" x $kafkaArchive -o"$tempDir" -y | Out-Null
    $tarFile = Get-ChildItem -Path $tempDir -Filter "*.tar" | Select-Object -First 1
    & "$env:ProgramFiles\7-Zip\7z.exe" x $tarFile.FullName -o"$InstallDir" -y | Out-Null
    
    # Create Kafka data and log directories
    $kafkaDataDir = Join-Path $DataDir "kafka"
    if (-not (Test-Path $kafkaDataDir)) {
        New-Item -ItemType Directory -Path $kafkaDataDir -Force | Out-Null
    }
    
    if (-not (Test-Path $LogDir)) {
        New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
    }
    
    # Create Kafka configuration
    $kafkaConfig = @"
# Broker ID
broker.id=$BrokerId

# Listeners
listeners=PLAINTEXT://0.0.0.0:$KafkaPort
advertised.listeners=PLAINTEXT://localhost:$KafkaPort

# Log settings
log.dirs=$($LogDir.Replace("\", "/"))
num.partitions=1
default.replication.factor=1
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

# ZooKeeper settings
zookeeper.connect=$ZooKeeperHost:$ZooKeeperPort
zookeeper.connection.timeout.ms=18000

# Socket settings
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

# Other settings
num.network.threads=3
num.io.threads=8
"@
    
    $kafkaConfigPath = Join-Path $kafkaDir "config\server.properties"
    Set-Content -Path $kafkaConfigPath -Value $kafkaConfig
    
    Write-Log "Kafka $Version installed successfully" "SUCCESS"
    return $true
}

function Create-WindowsService {
    param (
        [string]$ServiceName,
        [string]$DisplayName,
        [string]$Description,
        [string]$BinaryPath
    )
    
    Write-Log "Creating Windows service: $ServiceName..." "INFO"
    
    # Check if NSSM is installed
    $nssmPath = "$env:ProgramFiles\nssm\nssm.exe"
    if (-not (Test-Path $nssmPath)) {
        Write-Log "NSSM not found. Installing NSSM..." "INFO"
        try {
            choco install nssm -y
            Write-Log "NSSM installed successfully" "SUCCESS"
        } catch {
            Write-Log "Failed to install NSSM: $_" "ERROR"
            return $false
        }
    }
    
    # Check if service already exists
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Log "Service $ServiceName already exists. Removing..." "WARNING"
        & $nssmPath remove $ServiceName confirm
    }
    
    # Create service
    try {
        & $nssmPath install $ServiceName $BinaryPath
        & $nssmPath set $ServiceName DisplayName $DisplayName
        & $nssmPath set $ServiceName Description $Description
        & $nssmPath set $ServiceName Start SERVICE_AUTO_START
        
        Write-Log "Service $ServiceName created successfully" "SUCCESS"
        return $true
    } catch {
        Write-Log "Failed to create service $ServiceName: $_" "ERROR"
        return $false
    }
}

function Start-KafkaService {
    param (
        [string]$ServiceName
    )
    
    Write-Log "Starting service: $ServiceName..." "INFO"
    
    try {
        Start-Service -Name $ServiceName
        $service = Get-Service -Name $ServiceName
        if ($service.Status -eq "Running") {
            Write-Log "Service $ServiceName started successfully" "SUCCESS"
            return $true
        } else {
            Write-Log "Service $ServiceName failed to start" "ERROR"
            return $false
        }
    } catch {
        Write-Log "Failed to start service $ServiceName: $_" "ERROR"
        return $false
    }
}

# ============================================================================
# Main Script
# ============================================================================

# Show help if requested
if ($Help) {
    Show-Help
    exit 0
}

# Check if running as administrator
if (-not (Test-Administrator)) {
    Write-Host "This script must be run as Administrator. Please restart PowerShell as Administrator and try again." -ForegroundColor Red
    exit 1
}

Write-Host "Kafka Installation Script for Windows" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green
Write-Host ""

Write-Log "Starting Kafka installation with the following parameters:" "INFO"
Write-Log "Kafka Version: $KafkaVersion" "INFO"
Write-Log "Scala Version: $ScalaVersion" "INFO"
Write-Log "ZooKeeper Version: $ZooKeeperVersion" "INFO"
Write-Log "Java Version: $JavaVersion" "INFO"
Write-Log "Broker ID: $BrokerId" "INFO"
Write-Log "ZooKeeper Host: $ZooKeeperHost" "INFO"
Write-Log "ZooKeeper Port: $ZooKeeperPort" "INFO"
Write-Log "Kafka Port: $KafkaPort" "INFO"
Write-Log "Install Directory: $InstallDir" "INFO"
Write-Log "Data Directory: $DataDir" "INFO"
Write-Log "Log Directory: $LogDir" "INFO"
Write-Log "Install ZooKeeper: $InstallZooKeeper" "INFO"
Write-Log "Install Java: $InstallJava" "INFO"
Write-Log "Start Services: $StartServices" "INFO"
Write-Log "Create Services: $CreateServices" "INFO"

# Create installation directory if it doesn't exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Log "Created installation directory: $InstallDir" "INFO"
}

# Create data directory if it doesn't exist
if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Path $DataDir -Force | Out-Null
    Write-Log "Created data directory: $DataDir" "INFO"
}

# Install Java if requested
if ($InstallJava) {
    $javaInstalled = Install-JavaWithChoco -Version $JavaVersion
    if (-not $javaInstalled) {
        Write-Log "Failed to install Java. Exiting..." "ERROR"
        exit 1
    }
}

# Install ZooKeeper if requested
if ($InstallZooKeeper) {
    $zooKeeperInstalled = Install-ZooKeeper -Version $ZooKeeperVersion -InstallDir $InstallDir -DataDir $DataDir
    if (-not $zooKeeperInstalled) {
        Write-Log "Failed to install ZooKeeper. Exiting..." "ERROR"
        exit 1
    }
}

# Install Kafka
$kafkaInstalled = Install-Kafka -Version $KafkaVersion -ScalaVersion $ScalaVersion -InstallDir $InstallDir -DataDir $DataDir -LogDir $LogDir -BrokerId $BrokerId -ZooKeeperHost $ZooKeeperHost -ZooKeeperPort $ZooKeeperPort -KafkaPort $KafkaPort
if (-not $kafkaInstalled) {
    Write-Log "Failed to install Kafka. Exiting..." "ERROR"
    exit 1
}

# Create Windows services if requested
if ($CreateServices) {
    if ($InstallZooKeeper) {
        $zooKeeperDir = Join-Path $InstallDir "apache-zookeeper-$ZooKeeperVersion-bin"
        $zooKeeperBin = Join-Path $zooKeeperDir "bin\zkServer.cmd"
        $zooKeeperServiceCreated = Create-WindowsService -ServiceName "ZooKeeper" -DisplayName "Apache ZooKeeper" -Description "Apache ZooKeeper coordination service" -BinaryPath $zooKeeperBin
        if (-not $zooKeeperServiceCreated) {
            Write-Log "Failed to create ZooKeeper service. Continuing..." "WARNING"
        }
    }
    
    $kafkaDir = Join-Path $InstallDir "kafka_$ScalaVersion-$KafkaVersion"
    $kafkaBin = Join-Path $kafkaDir "bin\windows\kafka-server-start.bat"
    $kafkaConfig = Join-Path $kafkaDir "config\server.properties"
    $kafkaServiceCreated = Create-WindowsService -ServiceName "Kafka" -DisplayName "Apache Kafka" -Description "Apache Kafka messaging system" -BinaryPath "$kafkaBin $kafkaConfig"
    if (-not $kafkaServiceCreated) {
        Write-Log "Failed to create Kafka service. Continuing..." "WARNING"
    }
}

# Start services if requested
if ($StartServices) {
    if ($InstallZooKeeper -and $CreateServices) {
        $zooKeeperServiceStarted = Start-KafkaService -ServiceName "ZooKeeper"
        if (-not $zooKeeperServiceStarted) {
            Write-Log "Failed to start ZooKeeper service. Continuing..." "WARNING"
        }
        
        # Wait for ZooKeeper to start
        Write-Log "Waiting for ZooKeeper to start..." "INFO"
        Start-Sleep -Seconds 10
    }
    
    if ($CreateServices) {
        $kafkaServiceStarted = Start-KafkaService -ServiceName "Kafka"
        if (-not $kafkaServiceStarted) {
            Write-Log "Failed to start Kafka service. Continuing..." "WARNING"
        }
    }
}

Write-Host ""
Write-Host "Kafka Installation Complete!" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green
Write-Host ""
Write-Host "Installation Details:" -ForegroundColor Cyan
Write-Host "  Kafka Version: $KafkaVersion" -ForegroundColor Cyan
Write-Host "  Installation Directory: $InstallDir" -ForegroundColor Cyan
Write-Host "  Data Directory: $DataDir" -ForegroundColor Cyan
Write-Host "  Log Directory: $LogDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Kafka Configuration:" -ForegroundColor Cyan
Write-Host "  Broker ID: $BrokerId" -ForegroundColor Cyan
Write-Host "  Kafka Port: $KafkaPort" -ForegroundColor Cyan
Write-Host "  ZooKeeper Connection: $ZooKeeperHost:$ZooKeeperPort" -ForegroundColor Cyan
Write-Host ""
Write-Host "Services:" -ForegroundColor Cyan
if ($CreateServices) {
    if ($InstallZooKeeper) {
        $zooKeeperService = Get-Service -Name "ZooKeeper" -ErrorAction SilentlyContinue
        $zooKeeperStatus = if ($zooKeeperService) { $zooKeeperService.Status } else { "Not Created" }
        Write-Host "  ZooKeeper Service: $zooKeeperStatus" -ForegroundColor Cyan
    }
    
    $kafkaService = Get-Service -Name "Kafka" -ErrorAction SilentlyContinue
    $kafkaStatus = if ($kafkaService) { $kafkaService.Status } else { "Not Created" }
    Write-Host "  Kafka Service: $kafkaStatus" -ForegroundColor Cyan
} else {
    Write-Host "  Services were not created. Use -CreateServices to create Windows services." -ForegroundColor Cyan
}
Write-Host ""
Write-Host "To start Kafka manually:" -ForegroundColor Yellow
$kafkaDir = Join-Path $InstallDir "kafka_$ScalaVersion-$KafkaVersion"
if ($InstallZooKeeper) {
    $zooKeeperDir = Join-Path $InstallDir "apache-zookeeper-$ZooKeeperVersion-bin"
    Write-Host "  ZooKeeper: $zooKeeperDir\bin\zkServer.cmd" -ForegroundColor Yellow
}
Write-Host "  Kafka: $kafkaDir\bin\windows\kafka-server-start.bat $kafkaDir\config\server.properties" -ForegroundColor Yellow
Write-Host ""
Write-Host "Installation log: $env:TEMP\kafka_install.log" -ForegroundColor Cyan
Write-Host ""
