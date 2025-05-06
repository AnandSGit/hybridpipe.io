#Requires -Version 5.0

<#
.SYNOPSIS
    Renews TLS certificates for HybridPipe messaging systems.

.DESCRIPTION
    This script checks and renews TLS certificates for various messaging systems used by HybridPipe.
    It will only renew certificates that are about to expire (within 30 days by default).

.PARAMETER ConfigFile
    Path to the configuration file. Default is "tls_config.json".

.PARAMETER Protocol
    Renew certificates only for the specified protocol. If not specified, all enabled protocols will be checked.

.EXAMPLE
    .\Renew-Certificates.ps1
    Checks and renews certificates for all enabled protocols using the default configuration file.

.EXAMPLE
    .\Renew-Certificates.ps1 -Protocol kafka
    Checks and renews certificates only for Kafka using the default configuration file.

.EXAMPLE
    .\Renew-Certificates.ps1 -ConfigFile "custom_config.json"
    Checks and renews certificates for all enabled protocols using a custom configuration file.

.NOTES
    Author: HybridPipe Team
    Last Updated: April 19, 2025
#>

param (
    [string]$ConfigFile = "tls_config.json",
    [string]$Protocol = ""
)

# Default values
$script:LogFile = "tls_renewal.log"
$script:OutputDir = "certs"
$script:DaysThreshold = 30  # Renew certificates if they expire within 30 days

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

    return $config
}

# Function to generate a subject string for OpenSSL
function Get-SubjectString {
    param (
        [string]$CommonName
    )

    return "/C=$script:Country/ST=$script:State/L=$script:Locality/O=$script:Organization/OU=$script:OrgUnit/CN=$CommonName/emailAddress=$script:Email"
}

# Function to check if a certificate is about to expire
function Test-CertificateExpiry {
    param (
        [string]$CertFile,
        [int]$DaysThreshold
    )

    if (-not (Test-Path $CertFile)) {
        Write-Log -Level "WARN" -Message "Certificate file '$CertFile' not found."
        return $true  # Assume certificate needs renewal if file not found
    }

    try {
        $cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2 $CertFile
        $expiryDate = $cert.NotAfter
        $daysRemaining = ($expiryDate - (Get-Date)).Days

        Write-Log -Level "INFO" -Message "Certificate '$CertFile' expires in $daysRemaining days."

        return $daysRemaining -le $DaysThreshold
    }
    catch {
        Write-Log -Level "WARN" -Message "Failed to check certificate expiry: $_"
        return $true  # Assume certificate needs renewal if check fails
    }
}

# Function to renew server certificate for a protocol
function Update-ServerCertificate {
    param (
        [string]$Protocol,
        [string]$Prefix
    )

    $outputDir = Join-Path $script:OutputDir $Protocol
    $caCert = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $caKey = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $serverKey = Join-Path $outputDir "$Prefix-Server-Key.pem"
    $serverCsr = Join-Path $outputDir "$Prefix-Server.csr"
    $serverCert = Join-Path $outputDir "$Prefix-Server-Cert.pem"

    Write-Log -Level "INFO" -Message "Renewing server certificate for $Protocol..."

    # Check if CA certificate exists
    if (-not (Test-Path $caCert) -or -not (Test-Path $caKey)) {
        Write-ErrorAndExit "CA certificate or key not found for $Protocol. Please generate new certificates."
    }

    # Generate new CSR using existing key
    $subject = Get-SubjectString -CommonName $script:ServerHostname
    $result = & openssl req -new -key $serverKey -out $serverCsr -subj $subject 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate server CSR for ${Protocol}: $result"
    }

    # Sign new certificate with CA
    $result = & openssl x509 -req -CA $caCert -CAkey $caKey -in $serverCsr -out $serverCert -days $script:Validity -CAcreateserial -passin "pass:$script:Password" 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to sign server certificate for ${Protocol}: $result"
    }

    Write-Log -Level "INFO" -Message "Server certificate for $Protocol renewed successfully."

    # For Kafka, update the keystore
    if ($Protocol -eq "kafka") {
        $serverKeystore = Join-Path $outputDir "$script:ServerHostname-server.keystore.jks"

        # Delete old certificate from keystore
        $result = & keytool -delete -alias $script:ServerHostname -keystore $serverKeystore -storepass $script:Password 2>&1

        if ($LASTEXITCODE -ne 0) {
            Write-Log -Level "WARN" -Message "Failed to delete old certificate from keystore. Continuing anyway."
        }

        # Import new certificate into keystore
        $result = & keytool -import -alias $script:ServerHostname -file $serverCert -keystore $serverKeystore -storepass $script:Password -noprompt 2>&1

        if ($LASTEXITCODE -ne 0) {
            Write-ErrorAndExit "Failed to import renewed certificate into keystore for Kafka: $result"
        }

        Write-Log -Level "INFO" -Message "Kafka server keystore updated successfully."
    }
}

# Function to renew client certificate for a protocol
function Update-ClientCertificate {
    param (
        [string]$Protocol,
        [string]$Prefix
    )

    $outputDir = Join-Path $script:OutputDir $Protocol
    $caCert = Join-Path $outputDir "$Prefix-CA-Cert.pem"
    $caKey = Join-Path $outputDir "$Prefix-CA-Key.pem"
    $clientKey = Join-Path $outputDir "$Prefix-Client-Key.pem"
    $clientCsr = Join-Path $outputDir "$Prefix-Client.csr"
    $clientCert = Join-Path $outputDir "$Prefix-Client-Cert.pem"

    Write-Log -Level "INFO" -Message "Renewing client certificate for $Protocol..."

    # Check if CA certificate exists
    if (-not (Test-Path $caCert) -or -not (Test-Path $caKey)) {
        Write-ErrorAndExit "CA certificate or key not found for $Protocol. Please generate new certificates."
    }

    # Generate new CSR using existing key
    $subject = Get-SubjectString -CommonName "$Prefix-Client"
    $result = & openssl req -new -key $clientKey -out $clientCsr -subj $subject 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to generate client CSR for ${Protocol}: $result"
    }

    # Sign new certificate with CA
    $result = & openssl x509 -req -CA $caCert -CAkey $caKey -in $clientCsr -out $clientCert -days $script:Validity -CAcreateserial -passin "pass:$script:Password" 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Failed to sign client certificate for ${Protocol}: $result"
    }

    Write-Log -Level "INFO" -Message "Client certificate for $Protocol renewed successfully."
}

# Function to renew certificates for a specific protocol
function Update-ProtocolCertificates {
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
    $outputDir = Join-Path $script:OutputDir $Protocol
    $serverCert = Join-Path $outputDir "$prefix-Server-Cert.pem"
    $clientCert = Join-Path $outputDir "$prefix-Client-Cert.pem"

    Write-Log -Level "INFO" -Message "Checking certificates for $Protocol..."

    # Check if server certificate is about to expire
    if (Test-CertificateExpiry -CertFile $serverCert -DaysThreshold $script:DaysThreshold) {
        Update-ServerCertificate -Protocol $Protocol -Prefix $prefix
    }
    else {
        Write-Log -Level "INFO" -Message "Server certificate for $Protocol is still valid."
    }

    # Check if client certificate is about to expire
    if (Test-CertificateExpiry -CertFile $clientCert -DaysThreshold $script:DaysThreshold) {
        Update-ClientCertificate -Protocol $Protocol -Prefix $prefix
    }
    else {
        Write-Log -Level "INFO" -Message "Client certificate for $Protocol is still valid."
    }

    Write-Log -Level "INFO" -Message "Certificate renewal check for $Protocol completed."
}

# Function to renew certificates for all enabled protocols
function Update-AllCertificates {
    param (
        [PSObject]$Config
    )

    $protocols = $Config.protocols | Get-Member -MemberType NoteProperty | Select-Object -ExpandProperty Name

    foreach ($protocolName in $protocols) {
        if ($Protocol -and $protocolName -ne $Protocol) {
            Write-Log -Level "INFO" -Message "Skipping $protocolName (not specified in command line)"
            continue
        }

        Update-ProtocolCertificates -Protocol $protocolName -Config $Config
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
    Set-Content -Path $script:LogFile -Value "# TLS Certificate Renewal Log - $(Get-Date)"

    Write-Log -Level "INFO" -Message "Starting TLS certificate renewal check..."

    # Check requirements
    Test-Requirements

    # Read configuration
    $config = Read-Configuration

    # Renew certificates for all enabled protocols
    Update-AllCertificates -Config $config

    Write-Log -Level "INFO" -Message "TLS certificate renewal check completed successfully."
    Write-Host ""
    Write-Host "Certificate renewal check completed. See '$script:LogFile' for detailed logs."
}

# Run the main function
Main
