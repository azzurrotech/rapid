#!/bin/bash
# run.sh for rapid - Standalone RAPID AI analysis service
#
# Detects if rapid is already running locally and prompts:
#   use local / start new local / use remote
# Default: use any running local instance

set -e

RAPID_PORT="${RAPID_PORT:-8000}"

# Check if rapid is already running locally
is_local_running() {
    curl -s "http://127.0.0.1:$RAPID_PORT/api/health" > /dev/null 2>&1
    return $?
}

# Check if rapid remote is reachable
is_remote_running() {
    curl -s "http://${RAPID_REMOTE:-127.0.0.1:$RAPID_PORT}/api/health" > /dev/null 2>&1
    return $?
}

echo "=== rapid Service ==="
echo "Configuration:"
echo "  Local address: 127.0.0.1:$RAPID_PORT"

SELECTED=""

if is_local_running; then
    echo "✓ Local rapid is already running on 127.0.0.1:$RAPID_PORT"
    read -p "Use local instance? [Y/n/remote]: " choice
    case "$choice" in
        n|N)
            SELECTED="remote"
            ;;
        *)
            SELECTED="local"
            ;;
    esac
fi

if [ "$SELECTED" = "" ]; then
    if is_remote_running; then
        echo "✓ Remote rapid is reachable at ${RAPID_REMOTE:-127.0.0.1:$RAPID_PORT}"
        read -p "Use remote instance? [y/N/local]: " choice
        case "$choice" in
            y|Y)
                SELECTED="remote"
                ;;
            l|L)
                SELECTED="local"
                ;;
            *)
                SELECTED="local"
                ;;
        esac
    else
        echo "✗ No local or remote rapid running. Starting new local instance..."
        SELECTED="local"
    fi
fi

case "$SELECTED" in
    local)
        echo ">>> Starting new local rapid instance..."
        if [ -x "$(command -v rapid-cli)" ]; then
            rapid-cli serve -port "$RAPID_PORT" &
            echo "    PID: $!"
            echo "✓ rapid started locally on 127.0.0.1:$RAPID_PORT"
        else
            echo "✗ rapid-cli binary not found. Please build first: go build ./..."
            echo "  Or run: go run cmd/rapid-server/main.go -port $RAPID_PORT"
        fi
        ;;
    remote)
        echo ">>> Using remote rapid at ${RAPID_REMOTE:-127.0.0.1:$RAPID_PORT}"
        if [ -z "$RAPID_REMOTE" ]; then
            export RAPID_REMOTE="${RAPID_REMOTE:-127.0.0.1:$RAPID_PORT}"
            echo "✓ Remote set: $RAPID_REMOTE"
        fi
        ;;
    *)
        echo ">>> Starting new local rapid instance..."
        if [ -x "$(command -v rapid-cli)" ]; then
            rapid-cli serve -port "$RAPID_PORT" &
            echo "    PID: $!"
            echo "✓ rapid started locally on 127.0.0.1:$RAPID_PORT"
        else
            echo "✗ rapid-cli binary not found. Please build first: go build ./..."
            echo "  Or run: go run cmd/rapid-server/main.go -port $RAPID_PORT"
        fi
        ;;
esac

echo ""
echo "rapid service ready on port $RAPID_PORT"
echo "API endpoints:"
echo "  Health:     http://127.0.0.1:$RAPID_PORT/api/health"
echo "  Projects:   http://127.0.0.1:$RAPID_PORT/api/projects"
echo "  Wrappers:   http://127.0.0.1:$RAPID_PORT/api/wrappers"
echo "  Analyses:   http://127.0.0.1:$RAPID_PORT/api/analyses"
echo "  Integrations: http://127.0.0.1:$RAPID_PORT/api/integrations"
echo "  Share:      http://127.0.0.1:$RAPID_PORT/api/share"
echo "  Export:     http://127.0.0.1:$RAPID_PORT/api/export?table=projects&format=json"
