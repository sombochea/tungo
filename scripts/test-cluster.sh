#!/bin/bash

# TunGo Cluster Test Script
# This script helps test multi-server cluster functionality

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}╔═══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     TunGo Cluster Test Helper                     ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to start a server
start_server() {
    local port=$1
    local proxy_port=$2
    local log_file="logs/server-${port}.log"
    
    echo -e "${YELLOW}Starting server on control port ${port}, proxy port ${proxy_port}...${NC}"
    
    mkdir -p logs
    
    TUNGO_SERVER_CONTROL_PORT=${port} \
    TUNGO_SERVER_PORT=${proxy_port} \
    TUNGO_SERVER_LOG_LEVEL=info \
    TUNGO_SERVER_SUBDOMAIN_SUFFIX=localhost \
    go run cmd/server/main.go > "${log_file}" 2>&1 &
    
    local pid=$!
    echo ${pid} > "logs/server-${port}.pid"
    echo -e "${GREEN}✓ Server started on port ${port} (PID: ${pid})${NC}"
}

# Function to stop a server
stop_server() {
    local port=$1
    local pid_file="logs/server-${port}.pid"
    
    if [ -f "${pid_file}" ]; then
        local pid=$(cat "${pid_file}")
        echo -e "${YELLOW}Stopping server on port ${port} (PID: ${pid})...${NC}"
        kill ${pid} 2>/dev/null || true
        rm "${pid_file}"
        echo -e "${GREEN}✓ Server stopped${NC}"
    else
        echo -e "${RED}Server PID file not found${NC}"
    fi
}

# Function to stop all servers
stop_all_servers() {
    echo -e "${YELLOW}Stopping all servers...${NC}"
    for pid_file in logs/server-*.pid; do
        if [ -f "${pid_file}" ]; then
            local pid=$(cat "${pid_file}")
            kill ${pid} 2>/dev/null || true
            rm "${pid_file}"
        fi
    done
    echo -e "${GREEN}✓ All servers stopped${NC}"
}

# Function to show server logs
show_logs() {
    local port=$1
    local log_file="logs/server-${port}.log"
    
    if [ -f "${log_file}" ]; then
        echo -e "${YELLOW}=== Server ${port} Logs ===${NC}"
        tail -n 20 "${log_file}"
    else
        echo -e "${RED}Log file not found${NC}"
    fi
}

# Function to start client
start_client() {
    echo -e "${YELLOW}Starting client with cluster configuration...${NC}"
    
    # Create temporary config
    cat > logs/client-test.yaml <<EOF
server_cluster:
  - host: localhost
    port: 5555
  - host: localhost
    port: 5556
  - host: localhost
    port: 5557

local_host: localhost
local_port: 3000

connect_timeout: 10s
retry_interval: 5s
max_retries: 3

log_level: info
log_format: console
EOF

    echo -e "${GREEN}✓ Config created at logs/client-test.yaml${NC}"
    echo -e "${YELLOW}Run client with:${NC}"
    echo "  go run cmd/client/main.go --config logs/client-test.yaml --local-port 3000"
}

# Parse command
case "$1" in
    start-cluster)
        echo -e "${GREEN}Starting 3-server cluster...${NC}"
        start_server 5555 8080
        sleep 1
        start_server 5556 8081
        sleep 1
        start_server 5557 8082
        echo ""
        echo -e "${GREEN}✓ Cluster started!${NC}"
        echo "  Server 1: control=5555, proxy=8080"
        echo "  Server 2: control=5556, proxy=8081"
        echo "  Server 3: control=5557, proxy=8082"
        ;;
    
    start-server)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 start-server <control_port> <proxy_port>"
            exit 1
        fi
        start_server $2 $3
        ;;
    
    stop-server)
        if [ -z "$2" ]; then
            echo "Usage: $0 stop-server <control_port>"
            exit 1
        fi
        stop_server $2
        ;;
    
    stop-all)
        stop_all_servers
        ;;
    
    logs)
        if [ -z "$2" ]; then
            echo "Usage: $0 logs <control_port>"
            exit 1
        fi
        show_logs $2
        ;;
    
    logs-all)
        for log_file in logs/server-*.log; do
            if [ -f "${log_file}" ]; then
                port=$(basename "${log_file}" .log | cut -d'-' -f2)
                show_logs ${port}
                echo ""
            fi
        done
        ;;
    
    client-config)
        start_client
        ;;
    
    status)
        echo -e "${GREEN}=== Server Status ===${NC}"
        for pid_file in logs/server-*.pid; do
            if [ -f "${pid_file}" ]; then
                port=$(basename "${pid_file}" .pid | cut -d'-' -f2)
                pid=$(cat "${pid_file}")
                if ps -p ${pid} > /dev/null 2>&1; then
                    echo -e "${GREEN}✓ Server ${port} running (PID: ${pid})${NC}"
                else
                    echo -e "${RED}✗ Server ${port} not running (stale PID file)${NC}"
                    rm "${pid_file}"
                fi
            fi
        done
        ;;
    
    test-failover)
        echo -e "${GREEN}=== Testing Cluster Failover ===${NC}"
        echo ""
        echo "1. Start the cluster:"
        echo "   ./test-cluster.sh start-cluster"
        echo ""
        echo "2. Start the client (in another terminal):"
        echo "   ./test-cluster.sh client-config"
        echo "   go run cmd/client/main.go --config logs/client-test.yaml --local-port 3000"
        echo ""
        echo "3. Test failover by stopping servers one by one:"
        echo "   ./test-cluster.sh stop-server 5555"
        echo "   # Watch client rotate to 5556"
        echo "   ./test-cluster.sh stop-server 5556"
        echo "   # Watch client rotate to 5557"
        echo ""
        echo "4. Restart servers:"
        echo "   ./test-cluster.sh start-server 5555 8080"
        echo "   # Client stays on 5557"
        echo ""
        echo "5. Clean up:"
        echo "   ./test-cluster.sh stop-all"
        ;;
    
    clean)
        echo -e "${YELLOW}Cleaning up...${NC}"
        stop_all_servers
        rm -rf logs/
        echo -e "${GREEN}✓ Cleaned up${NC}"
        ;;
    
    *)
        echo "Usage: $0 {command}"
        echo ""
        echo "Commands:"
        echo "  start-cluster              Start 3-server cluster (ports 5555, 5556, 5557)"
        echo "  start-server <ctrl> <proxy> Start single server"
        echo "  stop-server <port>         Stop single server"
        echo "  stop-all                   Stop all servers"
        echo "  logs <port>                Show logs for server"
        echo "  logs-all                   Show logs for all servers"
        echo "  status                     Show server status"
        echo "  client-config              Generate client config for cluster"
        echo "  test-failover              Show failover testing instructions"
        echo "  clean                      Stop all and clean up"
        echo ""
        echo "Examples:"
        echo "  $0 start-cluster           # Start 3-server cluster"
        echo "  $0 client-config           # Generate client config"
        echo "  $0 stop-server 5555        # Stop first server (test failover)"
        echo "  $0 logs 5555               # View server logs"
        echo "  $0 stop-all                # Stop all servers"
        exit 1
        ;;
esac
