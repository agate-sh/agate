#!/bin/bash

# Test script to verify gemini readiness detection logic

echo "=== Testing Gemini Readiness Detection ==="
echo ""

# 1. Launch gemini in a test tmux session
echo "Step 1: Launching gemini in tmux session..."
tmux -L test new-session -d -s gemini_test
tmux -L test send-keys -t gemini_test "cd /Users/patrickerichsen/Git/agate-sh/agate" Enter
tmux -L test send-keys -t gemini_test "gemini" Enter

# 2. Capture initial content
initial_content=$(tmux -L test capture-pane -t gemini_test -p -e -J -S -)
echo "Step 2: Initial content captured"
echo "--- Initial content (first 200 chars) ---"
echo "$initial_content" | head -c 200
echo ""
echo ""

# 3. Poll until content changes (simulating our "wait for agent to start" logic)
echo "Step 3: Polling for content change (agent starting)..."
iteration=0
while true; do
    iteration=$((iteration + 1))
    sleep 0.5

    current_content=$(tmux -L test capture-pane -t gemini_test -p -e -J -S -)

    if [ "$current_content" != "$initial_content" ]; then
        echo "[$iteration] ✓ Content changed after ${iteration} iterations ($(($iteration * 500))ms)! Agent started."
        echo ""
        echo "--- Current content (first 300 chars) ---"
        echo "$current_content" | head -c 300
        echo ""
        echo ""
        break
    fi

    echo "[$iteration] Waiting for content to change..."

    # Timeout after 30 seconds
    if [ $iteration -gt 60 ]; then
        echo "❌ TIMEOUT: Content never changed after 30 seconds!"
        tmux -L test kill-session -t gemini_test
        exit 1
    fi
done

# 4. Now poll for stability (4 consecutive identical captures)
echo "Step 4: Beginning stability polling (need 4 consecutive identical captures)..."
stable_count=0
previous_content="$current_content"
stability_iteration=0

while [ $stable_count -lt 4 ]; do
    stability_iteration=$((stability_iteration + 1))
    sleep 0.5
    current_content=$(tmux -L test capture-pane -t gemini_test -p -e -J -S -)

    if [ "$current_content" = "$previous_content" ]; then
        stable_count=$((stable_count + 1))
        echo "[$stability_iteration] Stable ($stable_count/4)"
    else
        echo "[$stability_iteration] Content changed during stability polling! Resetting counter."
        stable_count=1
    fi

    previous_content="$current_content"

    # Timeout after 30 seconds of stability polling
    if [ $stability_iteration -gt 60 ]; then
        echo "❌ TIMEOUT: Screen never stabilized after 30 seconds!"
        tmux -L test kill-session -t gemini_test
        exit 1
    fi
done

echo ""
echo "✓ Screen stable after $stability_iteration stability checks! Agent ready."
echo ""
echo "--- Final stable content (last 400 chars) ---"
echo "$current_content" | tail -c 400
echo ""
echo ""

# 5. Now try to send a prompt
echo "Step 5: Sending test prompt to gemini..."
tmux -L test send-keys -t gemini_test -l "write a hello world function"
sleep 0.1
tmux -L test send-keys -t gemini_test Enter

# 6. Capture result after 2 seconds
echo "Waiting 2 seconds for prompt to be processed..."
sleep 2
final_content=$(tmux -L test capture-pane -t gemini_test -p -e -J -S -)

echo ""
echo "--- Content after prompt submission (last 600 chars) ---"
echo "$final_content" | tail -c 600
echo ""
echo ""

# Check if prompt was sent to gemini or shell
if echo "$final_content" | grep -q "write a hello world function"; then
    if echo "$final_content" | grep -q "gemini"; then
        echo "✓ SUCCESS: Prompt appears to have been sent to gemini"
    else
        echo "❌ FAILURE: Prompt was typed but may have gone to shell instead of gemini"
    fi
else
    echo "⚠ WARNING: Cannot find prompt in output"
fi

# Cleanup
echo ""
echo "Cleaning up test session..."
tmux -L test kill-session -t gemini_test
echo "Done!"
