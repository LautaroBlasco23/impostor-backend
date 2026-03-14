#!/bin/bash

# Colors
GREEN=$'\033[32m'
YELLOW=$'\033[33m'
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

# Handle data deletion for Docker
handle_docker_delete() {
    echo ""
    echo "  Delete Docker data?"
    echo ""
    echo "    1)  Yes — remove volumes (database data)"
    echo "    2)  No  — keep data (can restart anytime)"
    echo "    0)  Cancel"
    echo ""
    read -p "  Enter choice [0-2]: " delete_choice

    case $delete_choice in
        1)
            echo "  [•] Removing Docker volumes..."
            docker volume prune -f 2>/dev/null || true
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            echo "  ${GREEN}✅  Docker stopped. Data deleted.${RESET}"
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            ;;
        2)
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            echo "  ${GREEN}✅  Docker stopped. Data preserved.${RESET}"
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            ;;
        0)
            echo "  Cancelled"
            exit 0
            ;;
        *)
            echo "${RED}  Invalid choice${RESET}"
            handle_docker_delete
            ;;
    esac
}

# Handle data deletion for Local Dev
handle_dev_delete() {
    echo ""
    echo "  Delete Local Dev data?"
    echo ""
    echo "    1)  Yes — remove database volumes"
    echo "    2)  No  — keep data"
    echo "    0)  Cancel"
    echo ""
    read -p "  Enter choice [0-2]: " delete_choice

    case $delete_choice in
        1)
            echo "  [•] Removing database volumes..."
            docker volume prune -f 2>/dev/null || true
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            echo "  ${GREEN}✅  Local Dev stopped. Data deleted.${RESET}"
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            ;;
        2)
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            echo "  ${GREEN}✅  Local Dev stopped. Data preserved.${RESET}"
            echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
            ;;
        0)
            echo "  Cancelled"
            exit 0
            ;;
        *)
            echo "${RED}  Invalid choice${RESET}"
            handle_dev_delete
            ;;
    esac
}

# Main flow
print_banner

echo "  ${BOLD}Select an environment to stop:${RESET}"
echo ""
echo "    1)  Docker        — full stack (API + databases in containers)$(get_docker_badge)"
echo "    2)  Local Dev     — databases in Docker + native Go app$(get_dev_badge)"
echo "    3)  Local Docker  — full stack without nginx (localhost ports)$(get_local_badge)"
echo "    0)  Cancel"
echo ""

read -p "  Enter choice [0-3]: " choice

case $choice in
    1)
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        if docker_running; then
            echo "  Stopping Docker environment..."
            make full-docker-down
        else
            echo "  Docker is not running — skipping stop."
        fi
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        handle_docker_delete
        ;;
    2)
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        if dev_running; then
            echo "  Stopping Local Dev environment..."
            pkill -f 'air.*\.air\.toml'
            echo "  [•] Stopping databases..."
            make db-down
        else
            echo "  Local Dev is not running — skipping stop."
        fi
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        handle_dev_delete
        ;;
    3)
        echo ""
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        if local_running; then
            echo "  Stopping Local Docker environment..."
            make local-docker-down
        else
            echo "  Local Docker is not running — skipping stop."
        fi
        echo "  ${BOLD}────────────────────────────────────────────────────${RESET}"
        handle_docker_delete
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
