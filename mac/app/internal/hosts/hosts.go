package hosts

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	hostsFile   = "/etc/hosts"
	markerStart = "# K9-Web-Protection START"
	markerEnd   = "# K9-Web-Protection END"
)

// readHosts reads /etc/hosts — tries direct read first, falls back to `cat`
// in case the Wails sandbox restricts direct file access.
func readHosts() (string, error) {
	data, err := os.ReadFile(hostsFile)
	if err == nil {
		return string(data), nil
	}
	// Fallback: run cat via shell (not sandboxed)
	out, err2 := exec.Command("cat", hostsFile).Output()
	if err2 != nil {
		return "", err
	}
	return string(out), nil
}

func IsActive() bool {
	content, err := readHosts()
	if err != nil {
		return false
	}
	return strings.Contains(content, markerStart)
}

func Install(domains []string) error {
	content, err := readHosts()
	if err != nil {
		return err
	}

	// Remove existing K9 block if present
	if strings.Contains(content, markerStart) {
		start := strings.Index(content, markerStart)
		end := strings.Index(content, markerEnd) + len(markerEnd)
		content = strings.TrimRight(content[:start], "\n") + "\n" +
			strings.TrimLeft(content[end:], "\n")
	}

	var block strings.Builder
	block.WriteString("\n" + markerStart + "\n")
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		block.WriteString("0.0.0.0 " + d + "\n")
		if !strings.HasPrefix(d, "www.") {
			block.WriteString("0.0.0.0 www." + d + "\n")
		}
	}
	block.WriteString(markerEnd + "\n")

	newContent := strings.TrimRight(content, "\n") + block.String()
	return writeWithPrivileges(newContent)
}

func Remove() error {
	content, err := readHosts()
	if err != nil {
		return err
	}
	if !strings.Contains(content, markerStart) {
		return nil
	}

	start := strings.Index(content, markerStart)
	end := strings.Index(content, markerEnd) + len(markerEnd)
	newContent := strings.TrimRight(content[:start], "\n") + "\n" +
		strings.TrimLeft(content[end:], "\n")

	return writeWithPrivileges(newContent)
}

// writeWithPrivileges writes content to /etc/hosts via osascript.
// This shows the standard macOS admin password dialog — no root shell needed.
func writeWithPrivileges(content string) error {
	// Write new content to a temp file the shell script can copy from
	tmp, err := os.CreateTemp("", "k9hosts-*.txt")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err = tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	// osascript prompts for the admin password, then runs the shell command as root
	shellCmd := fmt.Sprintf(
		"cp %q /etc/hosts && dscacheutil -flushcache; killall -HUP mDNSResponder",
		tmp.Name(),
	)
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`,
		strings.ReplaceAll(shellCmd, `"`, `\"`),
	)

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not update /etc/hosts (admin required): %s", strings.TrimSpace(string(out)))
	}
	return nil
}
