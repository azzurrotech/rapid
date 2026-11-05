#!/bin/bash
# build-rapid.sh - Build and run the RAPID AI analysis front end
#
# Rapid integrates with the atp platform and can optionally connect to stenella.
#
# Usage:
#   ./build-rapid.sh           # build and start RAPID on default port
#   ./build-rapid.sh 8000    # start RAPID on port 8000
#   ./build-rapid.sh stenella # build and start stenella first, then RAPID
#   ./build-rapid.sh url      # connect to existing stenella/atp at given URL
#   ./build-rapid.sh host:port # specify custom host/port
#
# Automatically detects an open local port if none specified,
# otherwise accepts a URL as argument.

set -e

# Resolve script directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Parse arguments
PORT=""
STENELLA_URL=""
RUN_STENELLA=""

while [ $# -gt 0 ]; do
  case "$1" in
    stenella)
      RUN_STENELLA="true"
      shift
      ;;
    http://*|https://*)
      echo ">>> Connecting to provided URL: $1"
      STENELLA_URL="$1"
      shift
      ;;
    --port|-p)
      PORT="$2"
      shift 2
      ;;
    *)
      if echo "$1" | grep -qE '^[0-9]+$'; then
        PORT=":$1"
      else
        echo "ERROR: Expected a port number, stenella suboption, or URL, got: $1"
        exit 1
      fi
      shift
      ;;
  esac
done

# If stenella suboption requested, instruct user to start stenella separately
# and then run rapid pointing to it
if [ -n "$RUN_STENELLA" ]; then
  echo ">>> Stenella suboption requested"
  echo ">>> Please ensure STENELLA is running first"
  echo ">>> Then run: ./${SCRIPT_DIR##*/}/rapid-cli serve"
  # Don't attempt to auto-start stenella due to path complexities
  # Instead, set up for rapid to connect to a default stenella location
  STENELLA_URL="http://localhost:8085"
fi

# Find an open port if not provided
if [ -z "$PORT" ]; then
  for trial_port in 8000 8080 8081 8082 3000; do
    if ! curl -s "http://localhost${trial_port}" > /dev/null 2>&1; then
      PORT=":$trial_port"
      echo ">>> Using available rapid port $trial_port"
      break
    fi
  done
  if [ -z "$PORT" ]; then
    echo ">>> No default port available, relying on Go defaults/env"
    PORT=""
  fi
fi

# Build the Go binary
echo ">>> Building RAPID CLI binary..."
RAPID_MAIN="$SCRIPT_DIR/cmd/rapid-cli/main.go"
go build -o "$SCRIPT_DIR/rapid-cli" "$RAPID_MAIN"

# Verify binary was created
if [ ! -f "$SCRIPT_DIR/rapid-cli" ]; then
  echo "ERROR: Failed to build RAPID binary"
  exit 1
fi
chmod +x "$SCRIPT_DIR/rapid-cli"

echo ">>> Starting RAPID..."
if [ -n "$STENELLA_URL" ]; then
  echo ">>> Connecting to STENELLA at $STENELLA_URL"
  export RAPID_HOST="${STENELLA_URL#http://}"
  export RAPID_PORT="${PORT#:}"
else
  if [ -n "$PORT" ]; then
    echo ">>> Using port $PORT"
    export RAPID_HOST="localhost:${PORT#:}"
    export RAPID_PORT="${PORT#:}"
  else
    export RAPID_HOST="localhost:8000"
    export RAPID_PORT="8000"
  fi
fi

export RAPID_DATA_DIR="$SCRIPT_DIR/data"

"$SCRIPT_DIR/rapid-cli" serve