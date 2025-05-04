# TLS Certificate Management for HybridPipe

This directory contains scripts to generate and manage TLS certificates for secure communication with various messaging systems used by HybridPipe.

## Prerequisites

### For Linux/macOS
- OpenSSL
- Java Keytool (part of JDK)
- Bash shell
- jq (for JSON parsing in the new scripts)

### For Windows
- OpenSSL for Windows (can be installed via [Chocolatey](https://chocolatey.org/): `choco install openssl`)
- Java Keytool (part of JDK)
- PowerShell 5.0 or later

## Certificate Generation

### Configuration

The new scripts use a JSON configuration file (`tls_config.json`) to customize the certificate generation process. You can modify this file to change:

- Common settings (password, validity period, organization details)
- Protocol-specific settings (enabled/disabled, prefix, ports)
- Output directory and log file

### Linux/macOS

1. Make the script executable:
   ```bash
   chmod +x generate_tls_certs.sh
   ```

2. Run the script in interactive mode:
   ```bash
   ./generate_tls_certs.sh
   ```

3. Or run in non-interactive mode:
   ```bash
   ./generate_tls_certs.sh --non-interactive
   ```

4. You can also specify a custom configuration file:
   ```bash
   ./generate_tls_certs.sh --config custom_config.json
   ```

### Windows

1. Open PowerShell as Administrator.

2. Set execution policy to allow running scripts (if not already set):
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

3. Run the script:
   ```powershell
   .\Generate-TLSCerts.ps1
   ```

4. You can also specify a custom configuration file:
   ```powershell
   .\Generate-TLSCerts.ps1 -ConfigFile custom_config.json
   ```

## Certificate Renewal

The repository includes scripts to check and renew certificates that are about to expire (within 30 days by default).

### Linux/macOS

1. Make the script executable:
   ```bash
   chmod +x renew_certificates.sh
   ```

2. Run the script to check and renew all certificates:
   ```bash
   ./renew_certificates.sh
   ```

3. Or renew certificates for a specific protocol:
   ```bash
   ./renew_certificates.sh --protocol kafka
   ```

### Windows

1. Run the script to check and renew all certificates:
   ```powershell
   .\Renew-Certificates.ps1
   ```

2. Or renew certificates for a specific protocol:
   ```powershell
   .\Renew-Certificates.ps1 -Protocol kafka
   ```

## Legacy Scripts

The repository still includes the original certificate generation scripts for backward compatibility:

- `maketlsconfig.sh` for Linux/macOS
- `maketlsconfig.ps1` for Windows

These scripts have fewer features but may be simpler to use for basic certificate generation.

## Generated Files

The scripts generate the following files for each protocol in the `certs/<protocol>` directory:

### Common Files for All Protocols

- `<prefix>-CA-Key.pem`: CA private key
- `<prefix>-CA-Cert.pem`: CA certificate
- `<prefix>-Server-Key.pem`: Server private key
- `<prefix>-Server-Cert.pem`: Server certificate
- `<prefix>-Client-Key.pem`: Client private key
- `<prefix>-Client-Cert.pem`: Client certificate

### Additional Files for Kafka

- `<hostname>-server.keystore.jks`: Server keystore containing server private key and certificate
- `kafka.server.truststore.jks`: Server truststore containing CA certificate
- `kafka.client.truststore.jks`: Client truststore containing CA certificate

## Supported Protocols

The new scripts support the following protocols:

- Kafka
- NATS
- RabbitMQ
- Redis
- MQTT

You can enable or disable each protocol in the configuration file.

## Notes

- When prompted for the Common Name (CN) during certificate generation, enter the hostname that clients will use to connect to the server.
- The generated certificates are self-signed and suitable for development and testing. For production environments, consider using certificates signed by a trusted Certificate Authority (CA).
- The scripts generate separate certificates for each protocol to allow for different security configurations.
- All operations are logged to a log file for troubleshooting.

Last updated: April 19, 2025
