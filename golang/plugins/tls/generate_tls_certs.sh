#!/bin/bash

# Script to generate TLS certificates for HybridPipe messaging systems
# This script supports both interactive and non-interactive modes
# Usage: ./generate_tls_certs.sh [--non-interactive] [--config path/to/config.json]

# Default values
CONFIG_FILE="tls_config.json"
INTERACTIVE=true
LOG_FILE="tls_generation.log"
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
            --non-interactive)
                INTERACTIVE=false
                shift
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [--non-interactive] [--config path/to/config.json]"
                echo "  --non-interactive  Run in non-interactive mode (no prompts)"
                echo "  --config FILE      Use specified config file instead of default"
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

    # Create output directory if it doesn't exist
    mkdir -p "$OUTPUT_DIR"
}

# Function to pause and wait for user input in interactive mode
pause() {
    if [ "$INTERACTIVE" = true ]; then
        read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
    fi
}

# Function to generate a subject string for OpenSSL
generate_subject() {
    echo "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$1/emailAddress=$EMAIL"
}

# Function to generate CA certificates for a protocol
generate_ca_cert() {
    local protocol="$1"
    local prefix="$2"
    local output_dir="$OUTPUT_DIR/$protocol"
    
    mkdir -p "$output_dir"
    
    log "INFO" "Generating CA certificate for $protocol..."
    
    # Generate CA private key and certificate
    openssl req -new -x509 -nodes \
        -keyout "$output_dir/$prefix-CA-Key.pem" \
        -out "$output_dir/$prefix-CA-Cert.pem" \
        -days "$VALIDITY" \
        -subj "$(generate_subject "$prefix-CA")" \
        || error_exit "Failed to generate CA certificate for $protocol"
    
    log "INFO" "CA certificate for $protocol generated successfully."
}

# Function to generate server certificates for a protocol
generate_server_cert() {
    local protocol="$1"
    local prefix="$2"
    local output_dir="$OUTPUT_DIR/$protocol"
    
    log "INFO" "Generating server certificate for $protocol..."
    
    # Generate server private key and CSR
    openssl req -new -nodes \
        -keyout "$output_dir/$prefix-Server-Key.pem" \
        -out "$output_dir/$prefix-Server.csr" \
        -days "$VALIDITY" \
        -subj "$(generate_subject "$SERVER_HOSTNAME")" \
        || error_exit "Failed to generate server key and CSR for $protocol"
    
    # Sign server certificate with CA
    openssl x509 -req \
        -CA "$output_dir/$prefix-CA-Cert.pem" \
        -CAkey "$output_dir/$prefix-CA-Key.pem" \
        -in "$output_dir/$prefix-Server.csr" \
        -out "$output_dir/$prefix-Server-Cert.pem" \
        -days "$VALIDITY" \
        -CAcreateserial \
        -passin "pass:$PASSWORD" \
        || error_exit "Failed to sign server certificate for $protocol"
    
    log "INFO" "Server certificate for $protocol generated successfully."
}

# Function to generate client certificates for a protocol
generate_client_cert() {
    local protocol="$1"
    local prefix="$2"
    local output_dir="$OUTPUT_DIR/$protocol"
    
    log "INFO" "Generating client certificate for $protocol..."
    
    # Generate client private key and CSR
    openssl req -new -nodes \
        -keyout "$output_dir/$prefix-Client-Key.pem" \
        -out "$output_dir/$prefix-Client.csr" \
        -days "$VALIDITY" \
        -subj "$(generate_subject "$prefix-Client")" \
        || error_exit "Failed to generate client key and CSR for $protocol"
    
    # Sign client certificate with CA
    openssl x509 -req \
        -CA "$output_dir/$prefix-CA-Cert.pem" \
        -CAkey "$output_dir/$prefix-CA-Key.pem" \
        -in "$output_dir/$prefix-Client.csr" \
        -out "$output_dir/$prefix-Client-Cert.pem" \
        -days "$VALIDITY" \
        -CAcreateserial \
        -passin "pass:$PASSWORD" \
        || error_exit "Failed to sign client certificate for $protocol"
    
    log "INFO" "Client certificate for $protocol generated successfully."
}

# Function to generate Java keystores and truststores for Kafka
generate_kafka_keystores() {
    local protocol="kafka"
    local prefix=$(jq -r ".protocols.$protocol.prefix" "$CONFIG_FILE")
    local output_dir="$OUTPUT_DIR/$protocol"
    
    log "INFO" "Generating Java keystores and truststores for Kafka..."
    
    # Generate server keystore
    keytool -genkey -alias "$SERVER_HOSTNAME" \
        -keyalg RSA \
        -keystore "$output_dir/$SERVER_HOSTNAME-server.keystore.jks" \
        -keysize 2048 \
        -validity "$VALIDITY" \
        -storepass "$PASSWORD" \
        -keypass "$PASSWORD" \
        -dname "CN=$SERVER_HOSTNAME, OU=$ORG_UNIT, O=$ORGANIZATION, L=$LOCALITY, ST=$STATE, C=$COUNTRY" \
        -noprompt \
        || error_exit "Failed to generate server keystore for Kafka"
    
    # Generate server CSR
    keytool -certreq \
        -alias "$SERVER_HOSTNAME" \
        -keystore "$output_dir/$SERVER_HOSTNAME-server.keystore.jks" \
        -file "$output_dir/$SERVER_HOSTNAME-server.csr" \
        -storepass "$PASSWORD" \
        -keypass "$PASSWORD" \
        -noprompt \
        || error_exit "Failed to generate server CSR for Kafka"
    
    # Sign server certificate with CA
    openssl x509 -req \
        -CA "$output_dir/$prefix-CA-Cert.pem" \
        -CAkey "$output_dir/$prefix-CA-Key.pem" \
        -in "$output_dir/$SERVER_HOSTNAME-server.csr" \
        -out "$output_dir/$SERVER_HOSTNAME-server.cert.pem" \
        -days "$VALIDITY" \
        -CAcreateserial \
        -passin "pass:$PASSWORD" \
        || error_exit "Failed to sign server certificate for Kafka"
    
    # Import CA certificate into server keystore
    keytool -import \
        -alias "CARoot" \
        -file "$output_dir/$prefix-CA-Cert.pem" \
        -keystore "$output_dir/$SERVER_HOSTNAME-server.keystore.jks" \
        -storepass "$PASSWORD" \
        -keypass "$PASSWORD" \
        -noprompt \
        || error_exit "Failed to import CA certificate into server keystore for Kafka"
    
    # Import signed server certificate into server keystore
    keytool -import \
        -alias "$SERVER_HOSTNAME" \
        -file "$output_dir/$SERVER_HOSTNAME-server.cert.pem" \
        -keystore "$output_dir/$SERVER_HOSTNAME-server.keystore.jks" \
        -storepass "$PASSWORD" \
        -keypass "$PASSWORD" \
        -noprompt \
        || error_exit "Failed to import server certificate into server keystore for Kafka"
    
    # Create server truststore and import CA certificate
    keytool -import \
        -alias "CARoot" \
        -file "$output_dir/$prefix-CA-Cert.pem" \
        -keystore "$output_dir/kafka.server.truststore.jks" \
        -storepass "$PASSWORD" \
        -noprompt \
        || error_exit "Failed to create server truststore for Kafka"
    
    # Create client truststore and import CA certificate
    keytool -import \
        -alias "CARoot" \
        -file "$output_dir/$prefix-CA-Cert.pem" \
        -keystore "$output_dir/kafka.client.truststore.jks" \
        -storepass "$PASSWORD" \
        -noprompt \
        || error_exit "Failed to create client truststore for Kafka"
    
    log "INFO" "Java keystores and truststores for Kafka generated successfully."
}

# Function to generate certificates for a specific protocol
generate_protocol_certs() {
    local protocol="$1"
    local enabled=$(jq -r ".protocols.$protocol.enabled" "$CONFIG_FILE")
    
    if [ "$enabled" != "true" ]; then
        log "INFO" "Skipping $protocol (disabled in config)"
        return
    fi
    
    local prefix=$(jq -r ".protocols.$protocol.prefix" "$CONFIG_FILE")
    
    log "INFO" "Generating certificates for $protocol..."
    pause
    
    # Generate CA certificate
    generate_ca_cert "$protocol" "$prefix"
    
    # Generate server certificate
    generate_server_cert "$protocol" "$prefix"
    
    # Generate client certificate
    generate_client_cert "$protocol" "$prefix"
    
    # For Kafka, also generate Java keystores and truststores
    if [ "$protocol" = "kafka" ]; then
        generate_kafka_keystores
    fi
    
    log "INFO" "Certificate generation for $protocol completed."
    
    # Print summary of generated files
    echo ""
    echo "############## FOR ${protocol^^} ##"
    echo "Following files were generated in $OUTPUT_DIR/$protocol:"
    echo "###########################"
    echo "CA certificate: $prefix-CA-Cert.pem"
    echo "CA private key: $prefix-CA-Key.pem"
    echo "Server certificate: $prefix-Server-Cert.pem"
    echo "Server private key: $prefix-Server-Key.pem"
    echo "Client certificate: $prefix-Client-Cert.pem"
    echo "Client private key: $prefix-Client-Key.pem"
    
    if [ "$protocol" = "kafka" ]; then
        echo "Server java keystore: $SERVER_HOSTNAME-server.keystore.jks"
        echo "Server java truststore: kafka.server.truststore.jks"
        echo "Client java truststore: kafka.client.truststore.jks"
    fi
}

# Function to generate certificates for all enabled protocols
generate_all_certs() {
    # Get list of enabled protocols
    local protocols=$(jq -r '.protocols | keys[]' "$CONFIG_FILE")
    
    for protocol in $protocols; do
        generate_protocol_certs "$protocol"
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
    echo "# TLS Certificate Generation Log - $(date)" > "$LOG_FILE"
    
    log "INFO" "Starting TLS certificate generation..."
    
    # Check requirements
    check_requirements
    
    # Read configuration
    read_config
    
    # Generate certificates for all enabled protocols
    generate_all_certs
    
    log "INFO" "TLS certificate generation completed successfully."
    echo ""
    echo "All certificates have been generated in the '$OUTPUT_DIR' directory."
    echo "See '$LOG_FILE' for detailed logs."
}

# Run the main function
main "$@"
