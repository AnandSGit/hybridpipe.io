#!/bin/bash

# Script to renew TLS certificates for HybridPipe messaging systems
# Usage: ./renew_certificates.sh [--config path/to/config.json] [--protocol protocol_name]

# Default values
CONFIG_FILE="tls_config.json"
PROTOCOL=""
LOG_FILE="tls_renewal.log"
OUTPUT_DIR="certs"

# Function to log messages
log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

# Function to log errors and exit
error_exit() {
    log "ERROR" "$1"
    exit 1
}

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        error_exit "Required command '$1' not found. Please install it and try again."
    fi
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --protocol)
                PROTOCOL="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [--config path/to/config.json] [--protocol protocol_name]"
                echo "  --config FILE      Use specified config file instead of default"
                echo "  --protocol NAME    Renew certificates only for the specified protocol"
                echo "  --help             Show this help message"
                exit 0
                ;;
            *)
                error_exit "Unknown option: $1"
                ;;
        esac
    done
}

# Function to check if jq is installed
check_jq() {
    check_command "jq"
}

# Function to read configuration from JSON file
read_config() {
    if [ ! -f "$CONFIG_FILE" ]; then
        error_exit "Configuration file '$CONFIG_FILE' not found."
    fi

    # Check if the file is valid JSON
    if ! jq empty "$CONFIG_FILE" 2>/dev/null; then
        error_exit "Configuration file '$CONFIG_FILE' is not valid JSON."
    fi

    # Read common configuration
    PASSWORD=$(jq -r '.common.password' "$CONFIG_FILE")
    VALIDITY=$(jq -r '.common.validity' "$CONFIG_FILE")
    SERVER_HOSTNAME=$(jq -r '.common.server_hostname' "$CONFIG_FILE")
    COUNTRY=$(jq -r '.common.country' "$CONFIG_FILE")
    STATE=$(jq -r '.common.state' "$CONFIG_FILE")
    LOCALITY=$(jq -r '.common.locality' "$CONFIG_FILE")
    ORGANIZATION=$(jq -r '.common.organization' "$CONFIG_FILE")
    ORG_UNIT=$(jq -r '.common.organizational_unit' "$CONFIG_FILE")
    EMAIL=$(jq -r '.common.email' "$CONFIG_FILE")

    # Read output configuration
    OUTPUT_DIR=$(jq -r '.output.directory' "$CONFIG_FILE")
    LOG_FILE=$(jq -r '.output.log_file' "$CONFIG_FILE")
}

# Function to generate a subject string for OpenSSL
generate_subject() {
    echo "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$1/emailAddress=$EMAIL"
}

# Function to check if a certificate is about to expire
check_certificate_expiry() {
    local cert_file="$1"
    local days_threshold="$2"
    
    if [ ! -f "$cert_file" ]; then
        log "WARN" "Certificate file '$cert_file' not found."
        return 1
    }
    
    # Get certificate expiration date
    local expiry_date=$(openssl x509 -enddate -noout -in "$cert_file" | cut -d= -f2)
    local expiry_epoch=$(date -d "$expiry_date" +%s)
    local current_epoch=$(date +%s)
    local seconds_remaining=$((expiry_epoch - current_epoch))
    local days_remaining=$((seconds_remaining / 86400))
    
    log "INFO" "Certificate '$cert_file' expires in $days_remaining days."
    
    if [ "$days_remaining" -le "$days_threshold" ]; then
        return 0  # Certificate is about to expire
    else
        return 1  # Certificate is still valid
    fi
}

# Function to renew server certificate for a protocol
renew_server_cert() {
    local protocol="$1"
    local prefix="$2"
    local output_dir="$OUTPUT_DIR/$protocol"
    local ca_cert="$output_dir/$prefix-CA-Cert.pem"
    local ca_key="$output_dir/$prefix-CA-Key.pem"
    local server_key="$output_dir/$prefix-Server-Key.pem"
    local server_csr="$output_dir/$prefix-Server.csr"
    local server_cert="$output_dir/$prefix-Server-Cert.pem"
    
    log "INFO" "Renewing server certificate for $protocol..."
    
    # Check if CA certificate exists
    if [ ! -f "$ca_cert" ] || [ ! -f "$ca_key" ]; then
        error_exit "CA certificate or key not found for $protocol. Please generate new certificates."
    fi
    
    # Generate new CSR using existing key
    openssl req -new \
        -key "$server_key" \
        -out "$server_csr" \
        -subj "$(generate_subject "$SERVER_HOSTNAME")" \
        || error_exit "Failed to generate server CSR for $protocol"
    
    # Sign new certificate with CA
    openssl x509 -req \
        -CA "$ca_cert" \
        -CAkey "$ca_key" \
        -in "$server_csr" \
        -out "$server_cert" \
        -days "$VALIDITY" \
        -CAcreateserial \
        -passin "pass:$PASSWORD" \
        || error_exit "Failed to sign server certificate for $protocol"
    
    log "INFO" "Server certificate for $protocol renewed successfully."
    
    # For Kafka, update the keystore
    if [ "$protocol" = "kafka" ]; then
        local server_keystore="$output_dir/$SERVER_HOSTNAME-server.keystore.jks"
        
        # Delete old certificate from keystore
        keytool -delete \
            -alias "$SERVER_HOSTNAME" \
            -keystore "$server_keystore" \
            -storepass "$PASSWORD" \
            || log "WARN" "Failed to delete old certificate from keystore. Continuing anyway."
        
        # Import new certificate into keystore
        keytool -import \
            -alias "$SERVER_HOSTNAME" \
            -file "$server_cert" \
            -keystore "$server_keystore" \
            -storepass "$PASSWORD" \
            -noprompt \
            || error_exit "Failed to import renewed certificate into keystore for Kafka"
        
        log "INFO" "Kafka server keystore updated successfully."
    fi
}

# Function to renew client certificate for a protocol
renew_client_cert() {
    local protocol="$1"
    local prefix="$2"
    local output_dir="$OUTPUT_DIR/$protocol"
    local ca_cert="$output_dir/$prefix-CA-Cert.pem"
    local ca_key="$output_dir/$prefix-CA-Key.pem"
    local client_key="$output_dir/$prefix-Client-Key.pem"
    local client_csr="$output_dir/$prefix-Client.csr"
    local client_cert="$output_dir/$prefix-Client-Cert.pem"
    
    log "INFO" "Renewing client certificate for $protocol..."
    
    # Check if CA certificate exists
    if [ ! -f "$ca_cert" ] || [ ! -f "$ca_key" ]; then
        error_exit "CA certificate or key not found for $protocol. Please generate new certificates."
    fi
    
    # Generate new CSR using existing key
    openssl req -new \
        -key "$client_key" \
        -out "$client_csr" \
        -subj "$(generate_subject "$prefix-Client")" \
        || error_exit "Failed to generate client CSR for $protocol"
    
    # Sign new certificate with CA
    openssl x509 -req \
        -CA "$ca_cert" \
        -CAkey "$ca_key" \
        -in "$client_csr" \
        -out "$client_cert" \
        -days "$VALIDITY" \
        -CAcreateserial \
        -passin "pass:$PASSWORD" \
        || error_exit "Failed to sign client certificate for $protocol"
    
    log "INFO" "Client certificate for $protocol renewed successfully."
}

# Function to renew certificates for a specific protocol
renew_protocol_certs() {
    local protocol="$1"
    local enabled=$(jq -r ".protocols.$protocol.enabled" "$CONFIG_FILE")
    
    if [ "$enabled" != "true" ]; then
        log "INFO" "Skipping $protocol (disabled in config)"
        return
    fi
    
    local prefix=$(jq -r ".protocols.$protocol.prefix" "$CONFIG_FILE")
    local output_dir="$OUTPUT_DIR/$protocol"
    local server_cert="$output_dir/$prefix-Server-Cert.pem"
    local client_cert="$output_dir/$prefix-Client-Cert.pem"
    local days_threshold=30  # Renew certificates if they expire within 30 days
    
    log "INFO" "Checking certificates for $protocol..."
    
    # Check if server certificate is about to expire
    if check_certificate_expiry "$server_cert" "$days_threshold"; then
        renew_server_cert "$protocol" "$prefix"
    else
        log "INFO" "Server certificate for $protocol is still valid."
    fi
    
    # Check if client certificate is about to expire
    if check_certificate_expiry "$client_cert" "$days_threshold"; then
        renew_client_cert "$protocol" "$prefix"
    else
        log "INFO" "Client certificate for $protocol is still valid."
    fi
    
    log "INFO" "Certificate renewal check for $protocol completed."
}

# Function to renew certificates for all enabled protocols
renew_all_certs() {
    # Get list of enabled protocols
    local protocols=$(jq -r '.protocols | keys[]' "$CONFIG_FILE")
    
    for protocol in $protocols; do
        if [ -n "$PROTOCOL" ] && [ "$protocol" != "$PROTOCOL" ]; then
            log "INFO" "Skipping $protocol (not specified in command line)"
            continue
        fi
        
        renew_protocol_certs "$protocol"
    done
}

# Function to check required tools
check_requirements() {
    log "INFO" "Checking required tools..."
    
    check_command "openssl"
    check_command "keytool"
    check_command "jq"
    
    log "INFO" "All required tools are available."
}

# Main function
main() {
    # Parse command line arguments
    parse_args "$@"
    
    # Initialize log file
    echo "# TLS Certificate Renewal Log - $(date)" > "$LOG_FILE"
    
    log "INFO" "Starting TLS certificate renewal check..."
    
    # Check requirements
    check_requirements
    
    # Read configuration
    read_config
    
    # Renew certificates for all enabled protocols
    renew_all_certs
    
    log "INFO" "TLS certificate renewal check completed successfully."
    echo ""
    echo "Certificate renewal check completed. See '$LOG_FILE' for detailed logs."
}

# Run the main function
main "$@"
