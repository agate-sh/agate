package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Test parameters - use an existing session with trust prompt
	sessionName := "agate_agate_2134l_a858433d"
	paneIndex := 1

	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)

	fmt.Printf("=== Testing prompt handler on %s ===\n\n", target)

	// Test 1: Can we capture the pane?
	fmt.Println("Test 1: Capturing pane content...")
	captureCmd := exec.Command("tmux", "-L", "agate", "capture-pane", "-p", "-e", "-J", "-t", target)
	output, err := captureCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ CAPTURE FAILED: %v\nOutput: %s\n", err, string(output))
		return
	}
	fmt.Printf("✅ Capture succeeded (%d bytes)\n\n", len(output))

	content := string(output)

	// Test 2: Is the trust prompt visible?
	fmt.Println("Test 2: Checking for trust prompt...")
	hasTrustPrompt := strings.Contains(content, "Do you trust the files in this folder?")
	fmt.Printf("Trust prompt present: %v\n\n", hasTrustPrompt)

	// Test 3: Send test keystroke
	fmt.Println("Test 3: Sending test keystroke '.'...")
	sendCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, ".")
	if err := sendCmd.Run(); err != nil {
		fmt.Printf("❌ Send keystroke failed: %v\n", err)
		return
	}
	fmt.Println("✅ Test keystroke sent\n")

	// Test 4: Wait and capture again to see if keystroke appears
	fmt.Println("Test 4: Capturing again after 200ms to check for test keystroke...")
	time.Sleep(200 * time.Millisecond)
	captureCmd2 := exec.Command("tmux", "-L", "agate", "capture-pane", "-p", "-e", "-J", "-t", target)
	output2, err := captureCmd2.Output()
	if err != nil {
		fmt.Printf("❌ Second capture failed: %v\n", err)
		return
	}

	content2 := string(output2)
	lines := strings.Split(strings.TrimRight(content2, "\n"), "\n")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		fmt.Printf("Last line: %q\n", lastLine)
		fmt.Printf("Contains '.': %v\n", strings.Contains(lastLine, "."))

		// Also check if '.' appears anywhere
		fmt.Printf("'.' appears anywhere in content: %v\n\n", strings.Contains(content2, "."))
	}

	// Test 5: If trust prompt present, try to accept it
	if hasTrustPrompt {
		fmt.Println("Test 5: Attempting to accept trust prompt...")

		// Clean up test keystroke first
		fmt.Println("  - Cleaning up test keystroke with BSpace...")
		exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "BSpace").Run()
		time.Sleep(100 * time.Millisecond)

		// Send just Enter to accept (option 1 is already selected by default)
		fmt.Println("  - Sending Enter to accept (option 1 is pre-selected)...")
		acceptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "Enter")
		if err := acceptCmd.Run(); err != nil {
			fmt.Printf("❌ Failed to send accept: %v\n", err)
		} else {
			fmt.Println("✅ Sent '1' + Enter to accept\n")

			// Wait and check if prompt is gone (trust prompt processing takes 3-5+ seconds)
			fmt.Println("Test 6: Waiting for trust prompt to be processed (5 seconds)...")
			time.Sleep(5 * time.Second)
			fmt.Println("  - Checking if trust prompt disappeared...")
			captureCmd3 := exec.Command("tmux", "-L", "agate", "capture-pane", "-p", "-e", "-J", "-t", target)
			output3, _ := captureCmd3.Output()
			stillHasPrompt := strings.Contains(string(output3), "Do you trust the files in this folder?")

			if stillHasPrompt {
				fmt.Println("❌ Trust prompt STILL present after sending '1'")
				fmt.Println("\nCapturing current content to see what's there:")
				fmt.Println(string(output3)[:500]) // First 500 chars
			} else {
				fmt.Println("✅ Trust prompt successfully dismissed!")

				// Test 7: Now try test keystroke again to verify agent is ready
				fmt.Println("\nTest 7: Sending test keystroke again to verify agent ready...")
				exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, ".").Run()
				time.Sleep(200 * time.Millisecond)

				captureCmd4 := exec.Command("tmux", "-L", "agate", "capture-pane", "-p", "-e", "-J", "-t", target)
				output4, _ := captureCmd4.Output()
				content4 := string(output4)
				lines4 := strings.Split(strings.TrimRight(content4, "\n"), "\n")

				if len(lines4) > 0 {
					lastLine4 := lines4[len(lines4)-1]
					fmt.Printf("Last line after prompt dismissed: %q\n", lastLine4)

					if strings.Contains(lastLine4, ".") {
						fmt.Println("✅ Test keystroke visible! Agent is ready and accepting input")

						// Clean up
						exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "BSpace").Run()
					} else {
						fmt.Println("❌ Test keystroke NOT visible on last line")
						fmt.Printf("Full last 3 lines:\n")
						if len(lines4) >= 3 {
							for i := len(lines4) - 3; i < len(lines4); i++ {
								fmt.Printf("  %d: %q\n", i, lines4[i])
							}
						}
					}
				}
			}
		}
	}

	fmt.Println("\n=== Test complete ===")
}
