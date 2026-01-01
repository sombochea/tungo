#!/bin/bash

# TunGo Setup Script
# This script helps you get started quickly with TunGo

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  TunGo Setup Script${NC}"
    echo -e "${BLUE}================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}!${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Go version
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [[ $(echo -e "1.22\n$GO_VERSION" | sort -V | head -n1) == "1.22" ]]; then
            print_success "Go $GO_VERSION installed"
        else
            print_error "Go 1.22 or higher required. Found: $GO_VERSION"
            exit 1
        fi
    else
        print_error "Go is not installed. Please install Go 1.22 or higher."
        exit 1
    fi
    
    # Check Make
    if command -v make &> /dev/null; then
        print_success "Make is installed"
    else
        print_warning "Make is not installed. You can still build manually with 'go build'"
    fi
    
    # Check Docker (optional)
    if command -v docker &> /dev/null; then
        print_success "Docker is installed"
    else
        print_info "Docker is not installed (optional)"
    fi
    
    echo ""
}

build_project() {
    print_info "Building TunGo..."
    
    if command -v make &> /dev/null; then
        make build
    else
        mkdir -p bin
        go build -o bin/tungo-server ./cmd/server
        go build -o bin/tungo-client ./cmd/client
    fi
    
    if [ -f "bin/tungo-server" ] && [ -f "bin/tungo-client" ]; then
        print_success "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
    
    echo ""
}

setup_config() {
    print_info "Setting up configuration..."
    
    # Create data directory
    mkdir -p data
    print_success "Created data directory"
    
    # Copy config examples if they don't exist
    if [ ! -f "configs/server.yaml" ]; then
        if [ -f "configs/server.example.yaml" ]; then
            cp configs/server.example.yaml configs/server.yaml
            print_success "Created server config from example"
        fi
    else
        print_info "Server config already exists"
    fi
    
    if [ ! -f "configs/client.yaml" ]; then
        if [ -f "configs/client.example.yaml" ]; then
            cp configs/client.example.yaml configs/client.yaml
            print_success "Created client config from example"
        fi
    else
        print_info "Client config already exists"
    fi
    
    echo ""
}

run_tests() {
    print_info "Running tests..."
    
    if go test ./... -v; then
        print_success "All tests passed"
    else
        print_warning "Some tests failed (this is non-fatal)"
    fi
    
    echo ""
}

start_server() {
    print_info "Starting server..."
    
    # Kill any existing server
    pkill -f "tungo-server" 2>/dev/null || true
    
    # Start server in background
    ./bin/tungo-server > logs/server.log 2>&1 &
    SERVER_PID=$!
    
    # Wait for server to start
    sleep 2
    
    if ps -p $SERVER_PID > /dev/null; then
        print_success "Server started (PID: $SERVER_PID)"
        print_info "Server logs: tail -f logs/server.log"
    else
        print_error "Server failed to start. Check logs/server.log"
        exit 1
    fi
    
    echo ""
}

test_connection() {
    print_info "Testing server health..."
    
    # Wait a bit for server to be ready
    sleep 3
    
    if curl -s -f http://localhost:5000/health > /dev/null 2>&1; then
        print_success "Server health check passed"
        
        # Show health response
        HEALTH=$(curl -s http://localhost:5000/health)
        print_info "Server status: $HEALTH"
    else
        print_error "Server health check failed"
        print_info "Make sure server is running on port 5000"
    fi
    
    echo ""
}

show_next_steps() {
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}  Setup Complete!${NC}"
    echo -e "${GREEN}================================${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Start a local service to tunnel:"
    echo "   ${YELLOW}python3 -m http.server 8000${NC}"
    echo ""
    echo "2. In another terminal, start the client:"
    echo "   ${YELLOW}./bin/tungo-client --port 8000 --verbose${NC}"
    echo ""
    echo "3. Test your tunnel:"
    echo "   ${YELLOW}curl -H \"Host: <subdomain>.localhost\" http://localhost:8080/${NC}"
    echo ""
    echo "Useful commands:"
    echo "  ${BLUE}make run-server${NC}  - Run server with config"
    echo "  ${BLUE}make run-client${NC}  - Run client with defaults"
    echo "  ${BLUE}make test${NC}        - Run tests"
    echo "  ${BLUE}make clean${NC}       - Clean build artifacts"
    echo ""
    echo "Documentation:"
    echo "  ${BLUE}README.md${NC}       - Full documentation"
    echo "  ${BLUE}QUICKSTART.md${NC}   - Quick start guide"
    echo "  ${BLUE}SECURITY.md${NC}     - Security best practices"
    echo "  ${BLUE}DEVELOPMENT.md${NC}  - Developer guide"
    echo ""
}

cleanup() {
    print_info "Cleaning up..."
    pkill -f "tungo-server" 2>/dev/null || true
    print_success "Cleanup complete"
}

main() {
    print_header
    
    # Parse arguments
    SKIP_BUILD=false
    SKIP_SERVER=false
    RUN_TESTS=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-server)
                SKIP_SERVER=true
                shift
                ;;
            --run-tests)
                RUN_TESTS=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-build     Skip the build step"
                echo "  --skip-server    Don't start the server"
                echo "  --run-tests      Run tests before starting"
                echo "  --help           Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Create logs directory
    mkdir -p logs
    
    check_prerequisites
    
    if [ "$SKIP_BUILD" = false ]; then
        build_project
    fi
    
    setup_config
    
    if [ "$RUN_TESTS" = true ]; then
        run_tests
    fi
    
    if [ "$SKIP_SERVER" = false ]; then
        start_server
        test_connection
    fi
    
    show_next_steps
}

# Trap for cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"
