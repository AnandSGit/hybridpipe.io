#!/bin/bash
# ============================================================================
# Kafka Installation Script for Linux/macOS
# ============================================================================
# This script automates the installation and configuration of Apache Kafka
# and its dependencies (Java, ZooKeeper) on Linux and macOS systems.
#
# Author: HybridPipe.io
# Date: April 20, 2025
# Version: 1.0
# ============================================================================

# Default configuration values
KAFKA_VERSION="3.5.1"
SCALA_VERSION="2.13"
ZOOKEEPER_VERSION="3.8.1"
JAVA_VERSION="17"
BROKER_ID=0
ZOOKEEPER_HOST="localhost"
ZOOKEEPER_PORT=2181
KAFKA_PORT=9092
INSTALL_DIR="/opt/kafka"
DATA_DIR="/var/lib/kafka"
LOG_DIR="/var/log/kafka"
INSTALL_ZOOKEEPER=true
INSTALL_JAVA=true
START_SERVICES=true
CREATE_SERVICES=true
VERBOSE=false
SYSTEM_TYPE="systemd" # or "init" for older systems

# ============================================================================
# Function Definitions
# ============================================================================

# Print usage information
show_help() {
    echo "Kafka Installation Script for Linux/macOS"
    echo "===================================="
    echo ""
    echo "This script installs and configures Apache Kafka and its dependencies on Linux/macOS."
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -k, --kafka-version <version>     : Kafka version to install (default: $KAFKA_VERSION)"
    echo "  -s, --scala-version <version>     : Scala version for Kafka (default: $SCALA_VERSION)"
    echo "  -z, --zookeeper-version <version> : ZooKeeper version to install (default: $ZOOKEEPER_VERSION)"
    echo "  -j, --java-version <version>      : Java version to install (default: $JAVA_VERSION)"
    echo "  -b, --broker-id <id>              : Kafka broker ID (default: $BROKER_ID)"
    echo "  -h, --zookeeper-host <host>       : ZooKeeper host (default: $ZOOKEEPER_HOST)"
    echo "  -p, --zookeeper-port <port>       : ZooKeeper port (default: $ZOOKEEPER_PORT)"
    echo "  -l, --kafka-port <port>           : Kafka port (default: $KAFKA_PORT)"
    echo "  -i, --install-dir <path>          : Installation directory (default: $INSTALL_DIR)"
    echo "  -d, --data-dir <path>             : Data directory (default: $DATA_DIR)"
    echo "  -o, --log-dir <path>              : Log directory (default: $LOG_DIR)"
    echo "  -n, --no-zookeeper                : Skip ZooKeeper installation"
    echo "  -m, --no-java                     : Skip Java installation"
    echo "  -x, --no-start                    : Don't start services after installation"
    echo "  -c, --no-services                 : Don't create system services"
    echo "  -t, --init-system <type>          : Init system type (systemd or init, default: $SYSTEM_TYPE)"
    echo "  -v, --verbose                     : Enable verbose logging"
    echo "  --help                            : Show this help message"
    echo ""
    echo "Example:"
    echo "  $0 --kafka-version 3.5.1 --broker-id 1 --verbose"
    echo ""
}

# Log messages with timestamp and level
log() {
    local level="INFO"
    if [ $# -gt 1 ]; then
        level="$1"
        shift
    fi
    
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    local message="[$timestamp] [$level] $*"
    
    if [ "$level" = "ERROR" ]; then
        echo -e "\e[31m$message\e[0m"
    elif [ "$level" = "WARNING" ]; then
        echo -e "\e[33m$message\e[0m"
    elif [ "$level" = "SUCCESS" ]; then
        echo -e "\e[32m$message\e[0m"
    else
        if [ "$VERBOSE" = true ]; then
            echo -e "\e[36m$message\e[0m"
        fi
    fi
    
    # Also append to log file
    echo "$message" >> /tmp/kafka_install.log
}

# Check if running as root
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log "ERROR" "This script must be run as root. Please use sudo or run as root."
        exit 1
    fi
}

# Detect OS type (Linux or macOS)
detect_os() {
    if [ "$(uname)" = "Darwin" ]; then
        OS_TYPE="macos"
        log "Detected macOS system"
    else
        OS_TYPE="linux"
        # Detect Linux distribution
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            DISTRO="$ID"
            log "Detected Linux distribution: $DISTRO"
        else
            DISTRO="unknown"
            log "WARNING" "Unknown Linux distribution"
        fi
    fi
}

# Install Java
install_java() {
    log "Installing Java $JAVA_VERSION..."
    
    if [ "$OS_TYPE" = "macos" ]; then
        # Install Java on macOS using Homebrew
        if ! command -v brew &> /dev/null; then
            log "Homebrew not found. Installing Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        
        brew install openjdk@$JAVA_VERSION
        
        # Create symlink to make it available for system Java wrappers
        if [ -d "/Library/Java/JavaVirtualMachines" ]; then
            sudo ln -sfn "$(brew --prefix)/opt/openjdk@$JAVA_VERSION/libexec/openjdk.jdk" /Library/Java/JavaVirtualMachines/openjdk-$JAVA_VERSION.jdk
        fi
        
        # Set JAVA_HOME
        export JAVA_HOME="$(/usr/libexec/java_home -v $JAVA_VERSION)"
        echo "export JAVA_HOME=\"$JAVA_HOME\"" >> ~/.bash_profile
        
    else
        # Install Java on Linux
        case "$DISTRO" in
            "ubuntu"|"debian")
                apt-get update
                apt-get install -y openjdk-$JAVA_VERSION-jdk
                ;;
            "centos"|"rhel"|"fedora")
                yum install -y java-$JAVA_VERSION-openjdk-devel
                ;;
            "alpine")
                apk add --no-cache openjdk$JAVA_VERSION
                ;;
            *)
                log "ERROR" "Unsupported Linux distribution for automatic Java installation"
                log "Please install Java $JAVA_VERSION manually and try again"
                exit 1
                ;;
        esac
        
        # Set JAVA_HOME
        if [ -d "/usr/lib/jvm/java-$JAVA_VERSION-openjdk" ]; then
            export JAVA_HOME="/usr/lib/jvm/java-$JAVA_VERSION-openjdk"
        elif [ -d "/usr/lib/jvm/java-$JAVA_VERSION-openjdk-amd64" ]; then
            export JAVA_HOME="/usr/lib/jvm/java-$JAVA_VERSION-openjdk-amd64"
        else
            export JAVA_HOME=$(dirname $(dirname $(readlink -f $(which java))))
        fi
        
        echo "export JAVA_HOME=\"$JAVA_HOME\"" >> /etc/profile.d/java.sh
    fi
    
    log "SUCCESS" "Java $JAVA_VERSION installed successfully"
    log "JAVA_HOME set to $JAVA_HOME"
}

# Download a file from a URL
download_file() {
    local url="$1"
    local output_path="$2"
    
    log "Downloading $url to $output_path..."
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$output_path")"
    
    if command -v curl &> /dev/null; then
        curl -L -o "$output_path" "$url"
    elif command -v wget &> /dev/null; then
        wget -O "$output_path" "$url"
    else
        log "ERROR" "Neither curl nor wget found. Please install one of them and try again."
        return 1
    fi
    
    if [ -f "$output_path" ]; then
        log "SUCCESS" "Downloaded successfully to $output_path"
        return 0
    else
        log "ERROR" "Failed to download file to $output_path"
        return 1
    fi
}

# Extract a tar.gz archive
extract_archive() {
    local archive_path="$1"
    local destination_path="$2"
    
    log "Extracting $archive_path to $destination_path..."
    
    # Create destination directory if it doesn't exist
    mkdir -p "$destination_path"
    
    tar -xzf "$archive_path" -C "$destination_path" --strip-components=1
    
    if [ $? -eq 0 ]; then
        log "SUCCESS" "Extracted successfully to $destination_path"
        return 0
    else
        log "ERROR" "Failed to extract archive"
        return 1
    fi
}

# Install ZooKeeper
install_zookeeper() {
    log "Installing ZooKeeper $ZOOKEEPER_VERSION..."
    
    # Download ZooKeeper
    local zookeeper_url="https://dlcdn.apache.org/zookeeper/zookeeper-$ZOOKEEPER_VERSION/apache-zookeeper-$ZOOKEEPER_VERSION-bin.tar.gz"
    local zookeeper_archive="/tmp/apache-zookeeper-$ZOOKEEPER_VERSION-bin.tar.gz"
    
    if ! download_file "$zookeeper_url" "$zookeeper_archive"; then
        return 1
    fi
    
    # Create ZooKeeper directories
    local zookeeper_dir="$INSTALL_DIR/zookeeper"
    local zookeeper_data_dir="$DATA_DIR/zookeeper"
    
    mkdir -p "$zookeeper_dir"
    mkdir -p "$zookeeper_data_dir"
    
    # Extract ZooKeeper
    if ! extract_archive "$zookeeper_archive" "$zookeeper_dir"; then
        return 1
    fi
    
    # Create ZooKeeper configuration
    cat > "$zookeeper_dir/conf/zoo.cfg" << EOF
tickTime=2000
dataDir=$zookeeper_data_dir
clientPort=$ZOOKEEPER_PORT
initLimit=5
syncLimit=2
EOF
    
    # Set permissions
    chown -R $(id -u):$(id -g) "$zookeeper_dir"
    chown -R $(id -u):$(id -g) "$zookeeper_data_dir"
    
    log "SUCCESS" "ZooKeeper $ZOOKEEPER_VERSION installed successfully"
    return 0
}

# Install Kafka
install_kafka() {
    log "Installing Kafka $KAFKA_VERSION (Scala $SCALA_VERSION)..."
    
    # Download Kafka
    local kafka_url="https://dlcdn.apache.org/kafka/$KAFKA_VERSION/kafka_$SCALA_VERSION-$KAFKA_VERSION.tgz"
    local kafka_archive="/tmp/kafka_$SCALA_VERSION-$KAFKA_VERSION.tgz"
    
    if ! download_file "$kafka_url" "$kafka_archive"; then
        return 1
    fi
    
    # Create Kafka directories
    local kafka_dir="$INSTALL_DIR/kafka"
    local kafka_data_dir="$DATA_DIR/kafka"
    
    mkdir -p "$kafka_dir"
    mkdir -p "$kafka_data_dir"
    mkdir -p "$LOG_DIR"
    
    # Extract Kafka
    if ! extract_archive "$kafka_archive" "$kafka_dir"; then
        return 1
    fi
    
    # Create Kafka configuration
    cat > "$kafka_dir/config/server.properties" << EOF
# Broker ID
broker.id=$BROKER_ID

# Listeners
listeners=PLAINTEXT://0.0.0.0:$KAFKA_PORT
advertised.listeners=PLAINTEXT://localhost:$KAFKA_PORT

# Log settings
log.dirs=$LOG_DIR
num.partitions=1
default.replication.factor=1
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

# ZooKeeper settings
zookeeper.connect=$ZOOKEEPER_HOST:$ZOOKEEPER_PORT
zookeeper.connection.timeout.ms=18000

# Socket settings
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

# Other settings
num.network.threads=3
num.io.threads=8
EOF
    
    # Set permissions
    chown -R $(id -u):$(id -g) "$kafka_dir"
    chown -R $(id -u):$(id -g) "$kafka_data_dir"
    chown -R $(id -u):$(id -g) "$LOG_DIR"
    
    log "SUCCESS" "Kafka $KAFKA_VERSION installed successfully"
    return 0
}

# Create systemd service
create_systemd_service() {
    local service_name="$1"
    local description="$2"
    local exec_start="$3"
    local exec_stop="$4"
    local working_dir="$5"
    local user="$6"
    
    log "Creating systemd service: $service_name..."
    
    cat > "/etc/systemd/system/$service_name.service" << EOF
[Unit]
Description=$description
After=network.target

[Service]
Type=simple
User=$user
Group=$user
ExecStart=$exec_start
ExecStop=$exec_stop
WorkingDirectory=$working_dir
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable "$service_name"
    
    log "SUCCESS" "Service $service_name created successfully"
    return 0
}

# Create init.d service
create_init_service() {
    local service_name="$1"
    local description="$2"
    local exec_start="$3"
    local exec_stop="$4"
    local working_dir="$5"
    local user="$6"
    
    log "Creating init.d service: $service_name..."
    
    cat > "/etc/init.d/$service_name" << EOF
#!/bin/sh
### BEGIN INIT INFO
# Provides:          $service_name
# Required-Start:    \$remote_fs \$syslog
# Required-Stop:     \$remote_fs \$syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: $description
### END INIT INFO

DAEMON_USER=$user
DAEMON_PATH=$working_dir
DAEMON=$exec_start
DAEMON_ARGS=""
DAEMON_STOP=$exec_stop
NAME=$service_name
DESC="$description"
PID_FILE=/var/run/\$NAME.pid

# Exit if the package is not installed
[ -x "\$DAEMON" ] || exit 0

# Load the VERBOSE setting and other rcS variables
. /lib/init/vars.sh

# Define LSB log_* functions.
. /lib/lsb/init-functions

do_start() {
    start-stop-daemon --start --quiet --pidfile \$PID_FILE --exec \$DAEMON --test > /dev/null || return 1
    start-stop-daemon --start --quiet --chuid \$DAEMON_USER --background --make-pidfile --pidfile \$PID_FILE --exec \$DAEMON -- \$DAEMON_ARGS || return 2
}

do_stop() {
    start-stop-daemon --stop --quiet --retry=TERM/30/KILL/5 --pidfile \$PID_FILE --exec \$DAEMON_STOP
    RETVAL="\$?"
    [ "\$RETVAL" = 2 ] && return 2
    start-stop-daemon --stop --quiet --oknodo --retry=0/30/KILL/5 --exec \$DAEMON
    [ "\$?" = 2 ] && return 2
    rm -f \$PID_FILE
    return "\$RETVAL"
}

case "\$1" in
    start)
        log_daemon_msg "Starting \$DESC" "\$NAME"
        do_start
        case "\$?" in
            0|1) log_end_msg 0 ;;
            2) log_end_msg 1 ;;
        esac
        ;;
    stop)
        log_daemon_msg "Stopping \$DESC" "\$NAME"
        do_stop
        case "\$?" in
            0|1) log_end_msg 0 ;;
            2) log_end_msg 1 ;;
        esac
        ;;
    restart|force-reload)
        log_daemon_msg "Restarting \$DESC" "\$NAME"
        do_stop
        case "\$?" in
            0|1)
                do_start
                case "\$?" in
                    0) log_end_msg 0 ;;
                    1) log_end_msg 1 ;; # Old process is still running
                    *) log_end_msg 1 ;; # Failed to start
                esac
                ;;
            *)
                # Failed to stop
                log_end_msg 1
                ;;
        esac
        ;;
    status)
        status_of_proc "\$DAEMON" "\$NAME" && exit 0 || exit \$?
        ;;
    *)
        echo "Usage: \$0 {start|stop|restart|force-reload|status}" >&2
        exit 3
        ;;
esac
EOF
    
    chmod +x "/etc/init.d/$service_name"
    
    if [ "$OS_TYPE" = "linux" ]; then
        if [ "$DISTRO" = "ubuntu" ] || [ "$DISTRO" = "debian" ]; then
            update-rc.d "$service_name" defaults
        elif [ "$DISTRO" = "centos" ] || [ "$DISTRO" = "rhel" ] || [ "$DISTRO" = "fedora" ]; then
            chkconfig --add "$service_name"
            chkconfig "$service_name" on
        fi
    fi
    
    log "SUCCESS" "Service $service_name created successfully"
    return 0
}

# Create service (systemd or init.d)
create_service() {
    local service_name="$1"
    local description="$2"
    local exec_start="$3"
    local exec_stop="$4"
    local working_dir="$5"
    
    # Use current user for services
    local user=$(id -un)
    
    if [ "$SYSTEM_TYPE" = "systemd" ]; then
        create_systemd_service "$service_name" "$description" "$exec_start" "$exec_stop" "$working_dir" "$user"
    else
        create_init_service "$service_name" "$description" "$exec_start" "$exec_stop" "$working_dir" "$user"
    fi
}

# Start a service
start_service() {
    local service_name="$1"
    
    log "Starting service: $service_name..."
    
    if [ "$SYSTEM_TYPE" = "systemd" ]; then
        systemctl start "$service_name"
    else
        service "$service_name" start
    fi
    
    # Check if service is running
    if [ "$SYSTEM_TYPE" = "systemd" ]; then
        if systemctl is-active --quiet "$service_name"; then
            log "SUCCESS" "Service $service_name started successfully"
            return 0
        else
            log "ERROR" "Service $service_name failed to start"
            return 1
        fi
    else
        if service "$service_name" status > /dev/null; then
            log "SUCCESS" "Service $service_name started successfully"
            return 0
        else
            log "ERROR" "Service $service_name failed to start"
            return 1
        fi
    fi
}

# ============================================================================
# Parse Command Line Arguments
# ============================================================================

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -k|--kafka-version)
            KAFKA_VERSION="$2"
            shift 2
            ;;
        -s|--scala-version)
            SCALA_VERSION="$2"
            shift 2
            ;;
        -z|--zookeeper-version)
            ZOOKEEPER_VERSION="$2"
            shift 2
            ;;
        -j|--java-version)
            JAVA_VERSION="$2"
            shift 2
            ;;
        -b|--broker-id)
            BROKER_ID="$2"
            shift 2
            ;;
        -h|--zookeeper-host)
            ZOOKEEPER_HOST="$2"
            shift 2
            ;;
        -p|--zookeeper-port)
            ZOOKEEPER_PORT="$2"
            shift 2
            ;;
        -l|--kafka-port)
            KAFKA_PORT="$2"
            shift 2
            ;;
        -i|--install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -d|--data-dir)
            DATA_DIR="$2"
            shift 2
            ;;
        -o|--log-dir)
            LOG_DIR="$2"
            shift 2
            ;;
        -n|--no-zookeeper)
            INSTALL_ZOOKEEPER=false
            shift
            ;;
        -m|--no-java)
            INSTALL_JAVA=false
            shift
            ;;
        -x|--no-start)
            START_SERVICES=false
            shift
            ;;
        -c|--no-services)
            CREATE_SERVICES=false
            shift
            ;;
        -t|--init-system)
            SYSTEM_TYPE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# ============================================================================
# Main Script
# ============================================================================

# Show script banner
echo "Kafka Installation Script for Linux/macOS"
echo "===================================="
echo ""

# Initialize log file
echo "Kafka Installation Log - $(date)" > /tmp/kafka_install.log

# Check if running as root
check_root

# Detect OS type
detect_os

# Log configuration
log "Starting Kafka installation with the following parameters:"
log "Kafka Version: $KAFKA_VERSION"
log "Scala Version: $SCALA_VERSION"
log "ZooKeeper Version: $ZOOKEEPER_VERSION"
log "Java Version: $JAVA_VERSION"
log "Broker ID: $BROKER_ID"
log "ZooKeeper Host: $ZOOKEEPER_HOST"
log "ZooKeeper Port: $ZOOKEEPER_PORT"
log "Kafka Port: $KAFKA_PORT"
log "Install Directory: $INSTALL_DIR"
log "Data Directory: $DATA_DIR"
log "Log Directory: $LOG_DIR"
log "Install ZooKeeper: $INSTALL_ZOOKEEPER"
log "Install Java: $INSTALL_JAVA"
log "Start Services: $START_SERVICES"
log "Create Services: $CREATE_SERVICES"
log "System Type: $SYSTEM_TYPE"

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"
log "Created installation directory: $INSTALL_DIR"

# Create data directory if it doesn't exist
mkdir -p "$DATA_DIR"
log "Created data directory: $DATA_DIR"

# Install Java if requested
if [ "$INSTALL_JAVA" = true ]; then
    install_java
    if [ $? -ne 0 ]; then
        log "ERROR" "Failed to install Java. Exiting..."
        exit 1
    fi
fi

# Install ZooKeeper if requested
if [ "$INSTALL_ZOOKEEPER" = true ]; then
    install_zookeeper
    if [ $? -ne 0 ]; then
        log "ERROR" "Failed to install ZooKeeper. Exiting..."
        exit 1
    fi
fi

# Install Kafka
install_kafka
if [ $? -ne 0 ]; then
    log "ERROR" "Failed to install Kafka. Exiting..."
    exit 1
fi

# Create services if requested
if [ "$CREATE_SERVICES" = true ]; then
    if [ "$INSTALL_ZOOKEEPER" = true ]; then
        zookeeper_dir="$INSTALL_DIR/zookeeper"
        zookeeper_start="$zookeeper_dir/bin/zkServer.sh start-foreground"
        zookeeper_stop="$zookeeper_dir/bin/zkServer.sh stop"
        create_service "zookeeper" "Apache ZooKeeper coordination service" "$zookeeper_start" "$zookeeper_stop" "$zookeeper_dir"
    fi
    
    kafka_dir="$INSTALL_DIR/kafka"
    kafka_start="$kafka_dir/bin/kafka-server-start.sh $kafka_dir/config/server.properties"
    kafka_stop="$kafka_dir/bin/kafka-server-stop.sh"
    create_service "kafka" "Apache Kafka messaging system" "$kafka_start" "$kafka_stop" "$kafka_dir"
fi

# Start services if requested
if [ "$START_SERVICES" = true ]; then
    if [ "$INSTALL_ZOOKEEPER" = true ] && [ "$CREATE_SERVICES" = true ]; then
        start_service "zookeeper"
        
        # Wait for ZooKeeper to start
        log "Waiting for ZooKeeper to start..."
        sleep 10
    fi
    
    if [ "$CREATE_SERVICES" = true ]; then
        start_service "kafka"
    fi
fi

# Print installation summary
echo ""
echo "Kafka Installation Complete!"
echo "===================================="
echo ""
echo "Installation Details:"
echo "  Kafka Version: $KAFKA_VERSION"
echo "  Installation Directory: $INSTALL_DIR"
echo "  Data Directory: $DATA_DIR"
echo "  Log Directory: $LOG_DIR"
echo ""
echo "Kafka Configuration:"
echo "  Broker ID: $BROKER_ID"
echo "  Kafka Port: $KAFKA_PORT"
echo "  ZooKeeper Connection: $ZOOKEEPER_HOST:$ZOOKEEPER_PORT"
echo ""
echo "Services:"
if [ "$CREATE_SERVICES" = true ]; then
    if [ "$INSTALL_ZOOKEEPER" = true ]; then
        if [ "$SYSTEM_TYPE" = "systemd" ]; then
            zookeeper_status=$(systemctl is-active zookeeper)
        else
            zookeeper_status=$(service zookeeper status > /dev/null && echo "active" || echo "inactive")
        fi
        echo "  ZooKeeper Service: $zookeeper_status"
    fi
    
    if [ "$SYSTEM_TYPE" = "systemd" ]; then
        kafka_status=$(systemctl is-active kafka)
    else
        kafka_status=$(service kafka status > /dev/null && echo "active" || echo "inactive")
    fi
    echo "  Kafka Service: $kafka_status"
else
    echo "  Services were not created. Use --create-services to create system services."
fi
echo ""
echo "To start Kafka manually:"
if [ "$INSTALL_ZOOKEEPER" = true ]; then
    echo "  ZooKeeper: $INSTALL_DIR/zookeeper/bin/zkServer.sh start"
fi
echo "  Kafka: $INSTALL_DIR/kafka/bin/kafka-server-start.sh $INSTALL_DIR/kafka/config/server.properties"
echo ""
echo "Installation log: /tmp/kafka_install.log"
echo ""

exit 0
