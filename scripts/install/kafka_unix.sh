#!/bin/bash
#
# Kafka Installation Script for Linux/macOS
#
# This script installs Apache Kafka on Linux or macOS.
# It downloads Kafka, extracts it, and configures it.
#
# Usage:
#   ./kafka_unix.sh [options]
#
# Options:
#   --version <version>       Kafka version to install (default: 3.5.1)
#   --scala-version <version> Scala version to use (default: 2.13)
#   --install-dir <path>      Installation directory (default: /opt/kafka)
#   --zookeeper-port <port>   ZooKeeper port (default: 2181)
#   --kafka-port <port>       Kafka port (default: 9092)
#   --start-services          Start ZooKeeper and Kafka services after installation
#   --create-service          Create systemd services for ZooKeeper and Kafka
#   --verbose                 Show verbose output
#   --help                    Show this help message

# Default values
VERSION="3.5.1"
SCALA_VERSION="2.13"
INSTALL_DIR="/opt/kafka"
ZOOKEEPER_PORT=2181
KAFKA_PORT=9092
START_SERVICES=false
CREATE_SERVICE=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --scala-version)
            SCALA_VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --zookeeper-port)
            ZOOKEEPER_PORT="$2"
            shift 2
            ;;
        --kafka-port)
            KAFKA_PORT="$2"
            shift 2
            ;;
        --start-services)
            START_SERVICES=true
            shift
            ;;
        --create-service)
            CREATE_SERVICE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Kafka Installation Script for Linux/macOS"
            echo ""
            echo "Usage:"
            echo "  ./kafka_unix.sh [options]"
            echo ""
            echo "Options:"
            echo "  --version <version>       Kafka version to install (default: 3.5.1)"
            echo "  --scala-version <version> Scala version to use (default: 2.13)"
            echo "  --install-dir <path>      Installation directory (default: /opt/kafka)"
            echo "  --zookeeper-port <port>   ZooKeeper port (default: 2181)"
            echo "  --kafka-port <port>       Kafka port (default: 9092)"
            echo "  --start-services          Start ZooKeeper and Kafka services after installation"
            echo "  --create-service          Create systemd services for ZooKeeper and Kafka"
            echo "  --verbose                 Show verbose output"
            echo "  --help                    Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to log messages
log_message() {
    local message="$1"
    local type="${2:-INFO}"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo "[$timestamp] [$type] $message"
}

# Function to log verbose messages
log_verbose() {
    if [ "$VERBOSE" = true ]; then
        log_message "$1" "VERBOSE"
    fi
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if running as root
check_root() {
    if [ "$(id -u)" != "0" ]; then
        log_message "This script must be run as root" "ERROR"
        exit 1
    fi
}

# Function to detect the operating system
detect_os() {
    if [ "$(uname)" == "Darwin" ]; then
        echo "macos"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    else
        echo "unknown"
    fi
}

# Function to install Java
install_java() {
    log_message "Installing Java..."
    
    local os=$(detect_os)
    
    case "$os" in
        "macos")
            if command_exists brew; then
                brew install openjdk@17
            else
                log_message "Homebrew is not installed. Please install Homebrew first: https://brew.sh/" "ERROR"
                exit 1
            fi
            ;;
        "ubuntu"|"debian")
            apt-get update
            apt-get install -y openjdk-17-jdk
            ;;
        "centos"|"rhel"|"fedora")
            yum install -y java-17-openjdk
            ;;
        *)
            log_message "Unsupported operating system: $os" "ERROR"
            log_message "Please install Java 17 manually" "ERROR"
            exit 1
            ;;
    esac
    
    log_message "Java installation completed"
}

# Function to download a file
download_file() {
    local url="$1"
    local output_file="$2"
    
    log_message "Downloading $url to $output_file"
    
    if command_exists curl; then
        curl -L -o "$output_file" "$url"
    elif command_exists wget; then
        wget -O "$output_file" "$url"
    else
        log_message "Neither curl nor wget is installed" "ERROR"
        exit 1
    fi
    
    if [ $? -ne 0 ]; then
        log_message "Failed to download file" "ERROR"
        exit 1
    fi
    
    log_message "Download completed successfully"
}

# Function to create a directory
create_directory() {
    local path="$1"
    
    if [ ! -d "$path" ]; then
        log_message "Creating directory $path"
        mkdir -p "$path"
    fi
}

# Function to update a configuration file
update_config_file() {
    local file_path="$1"
    shift
    local properties=("$@")
    
    log_message "Updating configuration file $file_path"
    
    for prop in "${properties[@]}"; do
        local key=$(echo "$prop" | cut -d= -f1)
        local value=$(echo "$prop" | cut -d= -f2-)
        local pattern="^#?\s*$key\s*=.*"
        local replacement="$key=$value"
        local found=false
        
        if grep -q "$pattern" "$file_path"; then
            sed -i.bak "s|$pattern|$replacement|" "$file_path"
            found=true
        fi
        
        if [ "$found" = false ]; then
            echo "$replacement" >> "$file_path"
        fi
    done
    
    # Remove backup file
    rm -f "$file_path.bak"
}

# Function to create a systemd service
create_systemd_service() {
    local service_name="$1"
    local description="$2"
    local exec_start="$3"
    local working_dir="$4"
    local user="$5"
    
    log_message "Creating systemd service $service_name"
    
    local service_file="/etc/systemd/system/$service_name.service"
    
    cat > "$service_file" << EOF
[Unit]
Description=$description
After=network.target

[Service]
Type=simple
User=$user
ExecStart=$exec_start
WorkingDirectory=$working_dir
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable "$service_name"
    
    log_message "Service $service_name created successfully"
}

# Function to create a launchd service (macOS)
create_launchd_service() {
    local service_name="$1"
    local description="$2"
    local exec_start="$3"
    local working_dir="$4"
    
    log_message "Creating launchd service $service_name"
    
    local plist_file="/Library/LaunchDaemons/$service_name.plist"
    
    cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$service_name</string>
    <key>ProgramArguments</key>
    <array>
        <string>$exec_start</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$working_dir</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/$service_name.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/$service_name.err</string>
</dict>
</plist>
EOF
    
    chmod 644 "$plist_file"
    launchctl load "$plist_file"
    
    log_message "Service $service_name created successfully"
}

# Check if running as root
check_root

# Check if Java is installed
if ! command_exists java; then
    log_message "Java is not installed"
    install_java
fi

# Create installation directory
create_directory "$INSTALL_DIR"

# Download Kafka
KAFKA_VERSION="kafka_$SCALA_VERSION-$VERSION"
KAFKA_URL="https://downloads.apache.org/kafka/$VERSION/$KAFKA_VERSION.tgz"
KAFKA_ARCHIVE="/tmp/$KAFKA_VERSION.tgz"

download_file "$KAFKA_URL" "$KAFKA_ARCHIVE"

# Extract Kafka
log_message "Extracting Kafka archive"
tar -xzf "$KAFKA_ARCHIVE" -C "/tmp"

# Move Kafka to installation directory
log_message "Moving Kafka to installation directory"
cp -r "/tmp/$KAFKA_VERSION/"* "$INSTALL_DIR"

# Create data directories
ZOOKEEPER_DATA_DIR="$INSTALL_DIR/data/zookeeper"
KAFKA_DATA_DIR="$INSTALL_DIR/data/kafka"

create_directory "$ZOOKEEPER_DATA_DIR"
create_directory "$KAFKA_DATA_DIR"

# Update ZooKeeper configuration
ZOOKEEPER_CONFIG=(
    "dataDir=$ZOOKEEPER_DATA_DIR"
    "clientPort=$ZOOKEEPER_PORT"
    "maxClientCnxns=0"
)

update_config_file "$INSTALL_DIR/config/zookeeper.properties" "${ZOOKEEPER_CONFIG[@]}"

# Update Kafka configuration
KAFKA_CONFIG=(
    "broker.id=0"
    "listeners=PLAINTEXT://localhost:$KAFKA_PORT"
    "advertised.listeners=PLAINTEXT://localhost:$KAFKA_PORT"
    "log.dirs=$KAFKA_DATA_DIR"
    "zookeeper.connect=localhost:$ZOOKEEPER_PORT"
    "num.partitions=1"
    "default.replication.factor=1"
    "offsets.topic.replication.factor=1"
    "transaction.state.log.replication.factor=1"
    "transaction.state.log.min.isr=1"
)

update_config_file "$INSTALL_DIR/config/server.properties" "${KAFKA_CONFIG[@]}"

# Create shell scripts for starting ZooKeeper and Kafka
ZOOKEEPER_SCRIPT="$INSTALL_DIR/start-zookeeper.sh"
KAFKA_SCRIPT="$INSTALL_DIR/start-kafka.sh"

cat > "$ZOOKEEPER_SCRIPT" << EOF
#!/bin/bash
export KAFKA_HOME=$INSTALL_DIR
cd \$KAFKA_HOME
bin/zookeeper-server-start.sh config/zookeeper.properties
EOF

cat > "$KAFKA_SCRIPT" << EOF
#!/bin/bash
export KAFKA_HOME=$INSTALL_DIR
cd \$KAFKA_HOME
bin/kafka-server-start.sh config/server.properties
EOF

chmod +x "$ZOOKEEPER_SCRIPT"
chmod +x "$KAFKA_SCRIPT"

# Create services if requested
if [ "$CREATE_SERVICE" = true ]; then
    OS=$(detect_os)
    
    if [ "$OS" = "macos" ]; then
        create_launchd_service "org.apache.zookeeper" "Apache ZooKeeper" "$ZOOKEEPER_SCRIPT" "$INSTALL_DIR"
        create_launchd_service "org.apache.kafka" "Apache Kafka" "$KAFKA_SCRIPT" "$INSTALL_DIR"
    else
        create_systemd_service "zookeeper" "Apache ZooKeeper" "$ZOOKEEPER_SCRIPT" "$INSTALL_DIR" "root"
        create_systemd_service "kafka" "Apache Kafka" "$KAFKA_SCRIPT" "$INSTALL_DIR" "root"
    fi
fi

# Start services if requested
if [ "$START_SERVICES" = true ]; then
    if [ "$CREATE_SERVICE" = true ]; then
        OS=$(detect_os)
        
        if [ "$OS" = "macos" ]; then
            log_message "Starting ZooKeeper service"
            launchctl start org.apache.zookeeper
            
            log_message "Waiting for ZooKeeper to start"
            sleep 10
            
            log_message "Starting Kafka service"
            launchctl start org.apache.kafka
        else
            log_message "Starting ZooKeeper service"
            systemctl start zookeeper
            
            log_message "Waiting for ZooKeeper to start"
            sleep 10
            
            log_message "Starting Kafka service"
            systemctl start kafka
        fi
    else
        log_message "Starting ZooKeeper"
        nohup "$ZOOKEEPER_SCRIPT" > /dev/null 2>&1 &
        
        log_message "Waiting for ZooKeeper to start"
        sleep 10
        
        log_message "Starting Kafka"
        nohup "$KAFKA_SCRIPT" > /dev/null 2>&1 &
    fi
fi

# Clean up temporary files
rm -f "$KAFKA_ARCHIVE"
rm -rf "/tmp/$KAFKA_VERSION"

log_message "Kafka installation completed successfully"
log_message "ZooKeeper is configured to run on port $ZOOKEEPER_PORT"
log_message "Kafka is configured to run on port $KAFKA_PORT"
log_message "To start ZooKeeper: $ZOOKEEPER_SCRIPT"
log_message "To start Kafka: $KAFKA_SCRIPT"
