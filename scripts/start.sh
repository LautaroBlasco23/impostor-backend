#!/bin/bash

# Colors
GREEN=$'\033[32m'
RED=$'\033[31m'
BOLD=$'\033[1m'
RESET=$'\033[0m'

# Check if docker is running
docker_running() {
    docker compose ps --services --filter "status=running" 2>/dev/null | grep -q . 2>/dev/null
}

# Check if local dev is running
dev_running() {
    pgrep -f 'air.*\.air\.toml' >/dev/null 2>&1
}

# Check if local docker (no nginx) is running
local_running() {
    docker compose -f docker-compose.local.yml ps --services --filter "status=running" 2>/dev/null | grep -q . 2>/dev/null
}

# Print banner
print_banner() {
    echo ""
    echo "╔════════════════════════════════════════════════╗"
    echo "║     Impostor Backend Environment Manager      ║"
    echo "╚════════════════════════════════════════════════╝"
    echo ""
}

# Get running status badges
get_docker_badge() {
    docker_running && echo "  ${GREEN}(running)${RESET}" || echo ""
}

get_dev_badge() {
    dev_running && echo "  ${GREEN}(running)${RESET}" || echo ""
}

get_local_badge() {
    local_running && echo "  ${GREEN}(running)${RESET}" || echo ""
}

# Build prompt with created_at timestamp on existing images
show_build_prompt() {
    echo ""
    echo "  ${BOLD}Build new images?${RESET}"
    echo ""

    local images
    images=$(docker images --format "    • {{.Repository}}:{{.Tag}}  (created: {{.CreatedAt}})" --filter "reference=*impostor*" 2>/dev/null)

    echo "    1)  Yes — build fresh images"
    if [ -n "$images" ]; then
        echo "    2)  No  — use existing images:"
        echo "$images"
    else
        echo "    2)  No  — use existing images  (none found locally)"
    fi
    echo "    0)  Cancel"
    echo ""
    read -p "  Enter choice [0-2]: " build_choice

    case $build_choice in
        1) BUILD_FLAG="--build" ;;
        2) BUILD_FLAG="" ;;
        0) echo "  Cancelled"; exit 0 ;;
        *)
            echo "${RED}  Invalid choice${RESET}"
            exit 1
            ;;
    esac
}

# Check prerequisites
check_prerequisites() {
    case $1 in
        1) # Docker Compose
            if ! command -v docker &> /dev/null; then
                echo "${RED}❌ Docker is not installed${RESET}"
                return 1
            fi
            if ! docker info >/dev/null 2>&1; then
                echo "${RED}❌ Docker daemon is not running${RESET}"
                return 1
            fi
            if [ ! -f .env ]; then
                echo "${RED}❌ .env file not found. Copy from .env.example and configure.${RESET}"
                return 1
            fi
            ;;
        2) # Local Dev
            if ! command -v docker &> /dev/null; then
                echo "${RED}❌ Docker is not installed${RESET}"
                return 1
            fi
            if ! docker info >/dev/null 2>&1; then
                echo "${RED}❌ Docker daemon is not running${RESET}"
                return 1
            fi
            if ! command -v go &> /dev/null; then
                echo "${RED}❌ Go is not installed${RESET}"
                return 1
            fi
            ;;
        3) # Local Docker (no nginx)
            if ! command -v docker &> /dev/null; then
                echo "${RED}❌ Docker is not installed${RESET}"
                return 1
            fi
            if ! docker info >/dev/null 2>&1; then
                echo "${RED}❌ Docker daemon is not running${RESET}"
                return 1
            fi
            if [ ! -f .env ]; then
                echo "${RED}❌ .env file not found. Copy from .env.example and configure.${RESET}"
                return 1
            fi
            ;;
    esac
    return 0
}

# Main flow
print_banner

echo "  ${BOLD}Select an environment to start:${RESET}"
echo ""
echo "    1)  Docker        — full stack (API + databases in containers)$(get_docker_badge)"
echo "    2)  Local Dev     — databases in Docker + native Go app$(get_dev_badge)"
echo "    3)  Local Docker  — full stack without nginx (localhost ports)$(get_local_badge)"
echo "    0)  Cancel"
echo ""

read -p "  Enter choice [0-3]: " choice

case $choice in
    1)
        if ! check_prerequisites 1; then
            exit 1
        fi
        show_build_prompt
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  Starting Docker environment..."
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        make full-docker-up BUILD_FLAG="$BUILD_FLAG"
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  ${GREEN}✅  Docker environment started${RESET}"
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  API:  http://localhost:3000"
        echo "  Stop: make stop"
        ;;
    2)
        if ! check_prerequisites 2; then
            exit 1
        fi
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  Starting Local Dev environment..."
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        make dev
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  ${GREEN}✅  Local Dev started${RESET}"
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  Stop: make stop"
        ;;
    3)
        if ! check_prerequisites 3; then
            exit 1
        fi
        show_build_prompt
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  Starting Local Docker environment (no nginx)..."
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        make local-docker-up BUILD_FLAG="$BUILD_FLAG"
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  ${GREEN}✅  Local Docker environment started${RESET}"
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        echo "  API:  http://localhost:8080"
        echo "  Stop: make stop"
        ;;
    0)
        echo "  Cancelled"
        exit 0
        ;;
    *)
        echo "${RED}  Invalid choice${RESET}"
        exit 1
        ;;
esac
