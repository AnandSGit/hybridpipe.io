#Requires -Version 5.0

<#
.SYNOPSIS
    Generates TLS certificates for HybridPipe messaging systems.

.DESCRIPTION
    This script generates TLS certificates for various messaging systems used by HybridPipe.
    It supports both interactive and non-interactive modes, and can be configured using a JSON configuration file.

.PARAMETER NonInteractive
    Run in non-interactive mode (no prompts).

.PARAMETER ConfigFile
    Path to the configuration file. Default is "tls_config.json".

.EXAMPLE
    .\Generate-TLSCerts.ps1
    Runs the script in interactive mode with the default configuration file.

.EXAMPLE
    .\Generate-TLSCerts.ps1 -NonInteractive
    Runs the script in non-interactive mode with the default configuration file.

.EXAMPLE
    .\Generate-TLSCerts.ps1 -ConfigFile "custom_config.json"
    Runs the script with a custom configuration file.

.NOTES
    Author: HybridPipe Team
    Last Updated: April 19, 2025
#>

param (
    [switch]$NonInteractive,
    [string]$ConfigFile = "tls_config.json"
)

# Default values
$script:Interactive = -not $NonInteractive
$script:LogFile = "tls_generation.log"
$script:OutputDir = "certs"

# Function to log messages
function Write-Log {
    param (
        [string]$Level,
        [string]$Message
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    
    Write-Host $logMessage
    Add-Content -Path $script:LogFile -Value $logMessage
}

# Function to log errors and exit
function Write-ErrorAndExit {
    param (
        [string]$Message
    )
    
    Write-Log -Level "ERROR" -Message $Message
    exit 1
}

# Function to check if a command exists
function Test-Command {
    param (
        [string]$Command
    )
    
    if (-not (Get-Command $Command -ErrorAction SilentlyContinue)) {
        Write-ErrorAndExit "Required command '$Command' not found. Please install it and try again."
    }
}

# Function to pause and wait for user input in interactive mode
function Pause-Script {
    if ($script:Interactive) {
        Write-Host "Press any key to continue or Ctrl+C to exit..."
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    }
}

# Function to read configuration from JSON file
function Read-Configuration {
    if (-not (Test-Path $ConfigFile)) {
        Write-ErrorAndExit "Configuration file '$ConfigFile' not found."
    }
    
    try {
        $config = Get-Content -Path $ConfigFile -Raw | ConvertFrom-Json
    }
    catch {
        Write-ErrorAndExit "Failed to parse configuration file: $_"
    }
    
    # Read common configuration
    $script:Password = $config.common.password
    $script:Validity = $config.common.validity
    $script:ServerHostname = $config.common.server_hostname
    $script:Country = $config.common.country
    $script:State = $config.common.state
    $script:Locality = $config.common.locality
    $script:Organization = $config.common.organization
    $script:OrgUnit = $config.common.organizational_unit
    $script:Email = $config.common.email
    
    # Read output configuration
    $script:OutputDir = $config.output.directory
    $script:LogFile = $config.output.log_file
    
    # Create output directory if it doesn't exist
    if (-not (Test-Path $script:OutputDir)) {
        New-Item -Path $script:OutputDir -ItemType Directory | Out-Null
    }
    
    return $config
}

# Function to generate a subject string for OpenSSL
function Get-SubjectString {
    param (
        [string]$CommonName
    )
    
    return "/C=$script:Country/ST=$script:State/L=$script:Locality/O=$script:Organization/OU=$script:OrgUnit/CN=$CommonName/emailAddress=$script:Email"
}

# Function to generate CA certificates for a protocol
function New-CACertificate {
    param (
        [string]$Protocol,
        [string]$Prefix
    )
    
    $outputDir = Join-Path $script:OutputDir $Protocol
    
    if (-not (Test-Path $outputDir)) {
        New-Item -Path $outputDir -ItemType Directory | Out-Null
    }
    
    Write-Log -Level "INFO" -Message "Generating CA certificate for $Protocol..."
    
    # Generate CA private key and certificate
    $caKeyFile = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $caCertFile = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $subject = Get-SubjectString -CommonName "$Prefix-CA"
    
    $result = & openssl req -new -x509 -nodes -keyout $caKeyFile -out $caCertFile -days $script:Validity -subj $subject 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate CA certificate for $Protocol: $result"
    }
    
    Write-Log -Level "INFO" -Message "CA certificate for $Protocol generated successfully."
}

# Function to generate server certificates for a protocol
function New-ServerCertificate {
    param (
        [string]$Protocol,
        [string]$Prefix
    )
    
    $outputDir = Join-Path $script:OutputDir $Protocol
    
    Write-Log -Level "INFO" -Message "Generating server certificate for $Protocol..."
    
    # Generate server private key and CSR
    $serverKeyFile = Join-Path $outputDir "$Prefix-Server-Key.pem"
    $serverCsrFile = Join-Path $outputDir "$Prefix-Server.csr"
    $subject = Get-SubjectString -CommonName $script:ServerHostname
    
    $result = & openssl req -new -nodes -keyout $serverKeyFile -out $serverCsrFile -days $script:Validity -subj $subject 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate server key and CSR for $Protocol: $result"
    }
    
    # Sign server certificate with CA
    $caCertFile = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $caKeyFile = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $serverCertFile = Join-Path $outputDir "$Prefix-Server-Cert.pem"
    
    $result = & openssl x509 -req -CA $caCertFile -CAkey $caKeyFile -in $serverCsrFile -out $serverCertFile -days $script:Validity -CAcreateserial -passin "pass:$script:Password" 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to sign server certificate for $Protocol: $result"
    }
    
    Write-Log -Level "INFO" -Message "Server certificate for $Protocol generated successfully."
}

# Function to generate client certificates for a protocol
function New-ClientCertificate {
    param (
        [string]$Protocol,
        [string]$Prefix
    )
    
    $outputDir = Join-Path $script:OutputDir $Protocol
    
    Write-Log -Level "INFO" -Message "Generating client certificate for $Protocol..."
    
    # Generate client private key and CSR
    $clientKeyFile = Join-Path $outputDir "$Prefix-Client-Key.pem"
    $clientCsrFile = Join-Path $outputDir "$Prefix-Client.csr"
    $subject = Get-SubjectString -CommonName "$Prefix-Client"
    
    $result = & openssl req -new -nodes -keyout $clientKeyFile -out $clientCsrFile -days $script:Validity -subj $subject 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate client key and CSR for $Protocol: $result"
    }
    
    # Sign client certificate with CA
    $caCertFile = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $caKeyFile = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $clientCertFile = Join-Path $outputDir "$Prefix-Client-Cert.pem"
    
    $result = & openssl x509 -req -CA $caCertFile -CAkey $caKeyFile -in $clientCsrFile -out $clientCertFile -days $script:Validity -CAcreateserial -passin "pass:$script:Password" 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to sign client certificate for $Protocol: $result"
    }
    
    Write-Log -Level "INFO" -Message "Client certificate for $Protocol generated successfully."
}

# Function to generate Java keystores and truststores for Kafka
function New-KafkaKeystores {
    param (
        [string]$Protocol,
        [string]$Prefix
    )
    
    $outputDir = Join-Path $script:OutputDir $Protocol
    
    Write-Log -Level "INFO" -Message "Generating Java keystores and truststores for Kafka..."
    
    # Generate server keystore
    $serverKeystoreFile = Join-Path $outputDir "$script:ServerHostname-server.keystore.jks"
    $dname = "CN=$script:ServerHostname, OU=$script:OrgUnit, O=$script:Organization, L=$script:Locality, ST=$script:State, C=$script:Country"
    
    $result = & keytool -genkey -alias $script:ServerHostname -keyalg RSA -keystore $serverKeystoreFile -keysize 2048 -validity $script:Validity -storepass $script:Password -keypass $script:Password -dname $dname -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate server keystore for Kafka: $result"
    }
    
    # Generate server CSR
    $serverCsrFile = Join-Path $outputDir "$script:ServerHostname-server.csr"
    
    $result = & keytool -certreq -alias $script:ServerHostname -keystore $serverKeystoreFile -file $serverCsrFile -storepass $script:Password -keypass $script:Password -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate server CSR for Kafka: $result"
    }
    
    # Sign server certificate with CA
    $caCertFile = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $caKeyFile = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $serverCertFile = Join-Path $outputDir "$script:ServerHostname-server.cert.pem"
    
    $result = & openssl x509 -req -CA $caCertFile -CAkey $caKeyFile -in $serverCsrFile -out $serverCertFile -days $script:Validity -CAcreateserial -passin "pass:$script:Password" 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to sign server certificate for Kafka: $result"
    }
    
    # Import CA certificate into server keystore
    $result = & keytool -import -alias "CARoot" -file $caCertFile -keystore $serverKeystoreFile -storepass $script:Password -keypass $script:Password -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to import CA certificate into server keystore for Kafka: $result"
    }
    
    # Import signed server certificate into server keystore
    $result = & keytool -import -alias $script:ServerHostname -file $serverCertFile -keystore $serverKeystoreFile -storepass $script:Password -keypass $script:Password -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to import server certificate into server keystore for Kafka: $result"
    }
    
    # Create server truststore and import CA certificate
    $serverTruststoreFile = Join-Path $outputDir "kafka.server.truststore.jks"
    
    $result = & keytool -import -alias "CARoot" -file $caCertFile -keystore $serverTruststoreFile -storepass $script:Password -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to create server truststore for Kafka: $result"
    }
    
    # Create client truststore and import CA certificate
    $clientTruststoreFile = Join-Path $outputDir "kafka.client.truststore.jks"
    
    $result = & keytool -import -alias "CARoot" -file $caCertFile -keystore $clientTruststoreFile -storepass $script:Password -noprompt 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to create client truststore for Kafka: $result"
    }
    
    Write-Log -Level "INFO" -Message "Java keystores and truststores for Kafka generated successfully."
}

# Function to generate certificates for a specific protocol
function New-ProtocolCertificates {
    param (
        [string]$Protocol,
        [PSObject]$Config
    )
    
    $protocolConfig = $Config.protocols.$Protocol
    
    if (-not $protocolConfig.enabled) {
        Write-Log -Level "INFO" -Message "Skipping $Protocol (disabled in config)"
        return
    }
    
    $prefix = $protocolConfig.prefix
    
    Write-Log -Level "INFO" -Message "Generating certificates for $Protocol..."
    Pause-Script
    
    # Generate CA certificate
    New-CACertificate -Protocol $Protocol -Prefix $prefix
    
    # Generate server certificate
    New-ServerCertificate -Protocol $Protocol -Prefix $prefix
    
    # Generate client certificate
    New-ClientCertificate -Protocol $Protocol -Prefix $prefix
    
    # For Kafka, also generate Java keystores and truststores
    if ($Protocol -eq "kafka") {
        New-KafkaKeystores -Protocol $Protocol -Prefix $prefix
    }
    
    Write-Log -Level "INFO" -Message "Certificate generation for $Protocol completed."
    
    # Print summary of generated files
    Write-Host ""
    Write-Host "############## FOR $($Protocol.ToUpper()) ##"
    Write-Host "Following files were generated in $($script:OutputDir)\$Protocol:"
    Write-Host "###########################"
    Write-Host "CA certificate: $prefix-CA-Cert.pem"
    Write-Host "CA private key: $prefix-CA-Key.pem"
    Write-Host "Server certificate: $prefix-Server-Cert.pem"
    Write-Host "Server private key: $prefix-Server-Key.pem"
    Write-Host "Client certificate: $prefix-Client-Cert.pem"
    Write-Host "Client private key: $prefix-Client-Key.pem"
    
    if ($Protocol -eq "kafka") {
        Write-Host "Server java keystore: $script:ServerHostname-server.keystore.jks"
        Write-Host "Server java truststore: kafka.server.truststore.jks"
        Write-Host "Client java truststore: kafka.client.truststore.jks"
    }
}

# Function to generate certificates for all enabled protocols
function New-AllCertificates {
    param (
        [PSObject]$Config
    )
    
    $protocols = $Config.protocols | Get-Member -MemberType NoteProperty | Select-Object -ExpandProperty Name
    
    foreach ($protocol in $protocols) {
        New-ProtocolCertificates -Protocol $protocol -Config $Config
    }
}

# Function to check required tools
function Test-Requirements {
    Write-Log -Level "INFO" -Message "Checking required tools..."
    
    Test-Command -Command "openssl"
    Test-Command -Command "keytool"
    
    Write-Log -Level "INFO" -Message "All required tools are available."
}

# Main function
function Main {
    # Initialize log file
    Set-Content -Path $script:LogFile -Value "# TLS Certificate Generation Log - $(Get-Date)"
    
    Write-Log -Level "INFO" -Message "Starting TLS certificate generation..."
    
    # Check requirements
    Test-Requirements
    
    # Read configuration
    $config = Read-Configuration
    
    # Generate certificates for all enabled protocols
    New-AllCertificates -Config $config
    
    Write-Log -Level "INFO" -Message "TLS certificate generation completed successfully."
    Write-Host ""
    Write-Host "All certificates have been generated in the '$script:OutputDir' directory."
    Write-Host "See '$script:LogFile' for detailed logs."
}

# Run the main function
Main
