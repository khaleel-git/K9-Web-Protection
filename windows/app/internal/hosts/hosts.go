package hosts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	hostsFile   = `C:\Windows\System32\drivers\etc\hosts`
	markerStart = "# K9-Web-Protection START"
	markerEnd   = "# K9-Web-Protection END"
)

func readHosts() (string, error) {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
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

	if strings.Contains(content, markerStart) {
		start := strings.Index(content, markerStart)
		end := strings.Index(content, markerEnd) + len(markerEnd)
		content = strings.TrimRight(content[:start], "\r\n") + "\r\n" +
			strings.TrimLeft(content[end:], "\r\n")
	}

	var block strings.Builder
	block.WriteString("\r\n" + markerStart + "\r\n")
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		block.WriteString("0.0.0.0 " + d + "\r\n")
		if !strings.HasPrefix(d, "www.") {
			block.WriteString("0.0.0.0 www." + d + "\r\n")
		}
	}
	block.WriteString(markerEnd + "\r\n")

	newContent := strings.TrimRight(content, "\r\n") + block.String()
	return writeHosts(newContent)
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
	newContent := strings.TrimRight(content[:start], "\r\n") + "\r\n" +
		strings.TrimLeft(content[end:], "\r\n")
	return writeHosts(newContent)
}

func writeHosts(content string) error {
	// Direct write succeeds when the app is running as Administrator.
	if err := os.WriteFile(hostsFile, []byte(content), 0644); err == nil {
		flushDNS()
		return nil
	}
	// Fallback: elevate via a temp PowerShell script file (avoids all inline quoting issues).
	return writeWithElevation(content)
}

// writeWithElevation writes a .ps1 script to %TEMP% and runs it elevated.
// Using a script file instead of an inline -Command string avoids nested-quote nightmares.
func writeWithElevation(content string) error {
	tmp := filepath.Join(os.TempDir(), "k9hosts_temp.txt")
	script := filepath.Join(os.TempDir(), "k9hosts_install.ps1")

	if err := os.WriteFile(tmp, []byte(content), 0666); err != nil {
		return fmt.Errorf("could not write temp hosts file: %w", err)
	}
	defer os.Remove(tmp)

	// PowerShell single-quote escaping: '' inside a single-quoted string = literal '
	escapedTmp := strings.ReplaceAll(tmp, "'", "''")
	ps := "Copy-Item -Path '" + escapedTmp + "' -Destination 'C:\\Windows\\System32\\drivers\\etc\\hosts' -Force\r\nipconfig /flushdns\r\n"

	if err := os.WriteFile(script, []byte(ps), 0666); err != nil {
		return fmt.Errorf("could not write script file: %w", err)
	}
	defer os.Remove(script)

	// Pass the script path as -File argument — no inline quoting issues.
	// The outer single-quoted string safely contains double quotes around the path.
	psArgs := fmt.Sprintf(`-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%s"`, script)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`Start-Process powershell -ArgumentList '%s' -Verb RunAs -Wait`,
			strings.ReplaceAll(psArgs, "'", "''")))

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hosts file blocked — please right-click the app and choose 'Run as administrator'. (%s)",
			strings.TrimSpace(string(out)))
	}
	return nil
}

func flushDNS() {
	exec.Command("ipconfig", "/flushdns").Run()
}
