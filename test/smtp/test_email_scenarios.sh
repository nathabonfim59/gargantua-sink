#!/bin/bash

# Test configuration
SMTP_HOST="localhost"
SMTP_PORT="2525"
TEST_DIR="/tmp/gargantua-test"
STORAGE_DIR="$TEST_DIR/storage"
BINARY_PATH="$(pwd)/build/gargantua-sink"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    echo "Please run 'make build' first"
    exit 1
fi

# Check if port is available
if nc -z localhost $SMTP_PORT 2>/dev/null; then
    echo "Error: Port $SMTP_PORT is already in use"
    exit 1
fi

# Create test directories
mkdir -p "$TEST_DIR"
mkdir -p "$STORAGE_DIR"

# Start the SMTP server in background
echo "Starting Gargantua Sink SMTP server..."
$BINARY_PATH --port $SMTP_PORT --storage-path "$STORAGE_DIR" &
SERVER_PID=$!

# Wait for server to start and verify it's running
sleep 2
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "Error: Server failed to start"
    exit 1
fi

# Function to check if email exists
check_email() {
    local domain=$1
    local user=$2
    local direction=$3
    local pattern=$4
    local max_retries=5
    local retry=0
    local dir="$STORAGE_DIR/$domain/$user/$direction"
    
    echo "DEBUG: Checking directory: $dir"
    
    while [ $retry -lt $max_retries ]; do
        echo "Checking for email in $dir/ (attempt $((retry+1)))"
        if [ -d "$dir" ]; then
            echo "DEBUG: Directory exists"
            ls -la "$dir" || true
            if ls "$dir"/* >/dev/null 2>&1; then
                echo "DEBUG: Files found in directory"
                for file in "$dir"/*; do
                    echo "DEBUG: Checking file: $file"
                    if grep -q "$pattern" "$file"; then
                        echo "SUCCESS: Email with pattern '$pattern' found!"
                        return 0
                    else
                        echo "DEBUG: Pattern not found in file"
                        echo "DEBUG: File contents:"
                        cat "$file" || true
                    fi
                done
            else
                echo "DEBUG: No files in directory"
            fi
        else
            echo "DEBUG: Directory does not exist"
            ls -la "$STORAGE_DIR/$domain/$user" || true
        fi
        sleep 1
        ((retry++))
    done
    
    echo "ERROR: Email with pattern '$pattern' not found after $max_retries attempts!"
    return 1
}

echo "Running email tests..."

# Test 1: Simple email (incoming)
echo "Test 1: Simple email (incoming)"
swaks --server $SMTP_HOST --port $SMTP_PORT \
    --from sender@example.com \
    --to recipient@example.com \
    --header "Subject: Test Email" \
    --body "This is a test email" || { echo "Failed to send email"; exit 1; }
sleep 2
check_email "example.com" "recipient" "IN" "This is a test email"

# Test 2: Email with attachment (incoming)
echo "Test 2: Email with attachment (incoming)"
echo "Test attachment content" > "$TEST_DIR/attachment.txt"
swaks --server $SMTP_HOST --port $SMTP_PORT \
    --from sender@example.com \
    --to recipient@example.com \
    --header "Subject: Test Email with Attachment" \
    --attach "$TEST_DIR/attachment.txt" \
    --body "Email with attachment" || { echo "Failed to send email"; exit 1; }
sleep 2
check_email "example.com" "recipient" "IN" "attachment.txt"

# Test 3: Multiple recipients (incoming)
echo "Test 3: Multiple recipients (incoming)"
swaks --server $SMTP_HOST --port $SMTP_PORT \
    --from sender@example.com \
    --to "recipient1@test.com,recipient2@test.com" \
    --header "Subject: Multiple Recipients" \
    --body "Email to multiple recipients" || { echo "Failed to send email"; exit 1; }
sleep 2
check_email "test.com" "recipient1" "IN" "multiple recipients"
check_email "test.com" "recipient2" "IN" "multiple recipients"

# Test 4: HTML email (incoming)
echo "Test 4: HTML email (incoming)"
swaks --server $SMTP_HOST --port $SMTP_PORT \
    --from sender@example.com \
    --to recipient@example.com \
    --header "Subject: HTML Email" \
    --header "Content-Type: text/html" \
    --body "<h1>HTML Test</h1><p>This is an HTML email</p>" || { echo "Failed to send email"; exit 1; }
sleep 2
check_email "example.com" "recipient" "IN" "HTML Test"

# Test 5: Outgoing email
echo "Test 5: Outgoing email"
swaks --server $SMTP_HOST --port $SMTP_PORT \
    --from sender@example.com \
    --to recipient@example.com \
    --header "Subject: Outgoing Test" \
    --body "This is an outgoing test email" || { echo "Failed to send email"; exit 1; }
sleep 2

# Check both incoming and outgoing storage
check_email "example.com" "sender" "OUT" "outgoing test email"
check_email "example.com" "recipient" "IN" "outgoing test email"

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

echo "All tests completed!"
