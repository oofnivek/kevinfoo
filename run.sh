#!/bin/bash

# Find process ID listening on port 8080
PID=$(lsof -t -i:8080)

if [ -n "$PID" ]; then
    echo "Stopping existing process on port 8080 (PID: $PID)..."
    kill -9 $PID
    # Small sleep to ensure the port is released
    sleep 1
else
    echo "No process found on port 8080."
fi

echo "Starting application..."
# Ensure cargo is in PATH (standard for many local setups)
if [ -f "$HOME/.cargo/env" ]; then
    source "$HOME/.cargo/env"
fi

cargo run
