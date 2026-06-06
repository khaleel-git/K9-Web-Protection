package hosts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	markerStart = "# K10-Web-Protection START"
	markerEnd   = "# K10-Web-Protection END"

	ssMarkerStart = "# K10-SafeSearch START"
	ssMarkerEnd   = "# K10-SafeSearch END"
)

// hostsPath returns the Windows hosts file path, respecting %SystemRoot%.
func hostsPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, `System32\drivers\etc\hosts`)
}

// SafeSearch enforcement IPs published by Google / Microsoft for parental controls.
var safeSearchEntries = [][2]string{
	// Google — forcesafesearch.google.com
	{"216.239.38.120", "www.google.com"},
	{"216.239.38.120", "google.com"},
	// YouTube Restricted
	{"216.239.38.119", "www.youtube.com"},
	{"216.239.38.119", "m.youtube.com"},
	{"216.239.38.119", "youtubei.googleapis.com"},
	{"216.239.38.119", "youtube.googleapis.com"},
	{"216.239.38.119", "www.youtube-nocookie.com"},
	// Bing — strict.bing.com
	{"204.79.197.220", "www.bing.com"},
	{"204.79.197.220", "bing.com"},
}

// SetSafeSearch adds or removes the SafeSearch enforcement entries in the hosts file.
func SetSafeSearch(enabled bool) error {
	content, err := readHosts()
	if err != nil {
		return err
	}

	alreadyPresent := strings.Contains(content, ssMarkerStart)

	if enabled && alreadyPresent {
		return nil
	}
	if !enabled && !alreadyPresent {
		return nil
	}

	if alreadyPresent {
		start := strings.Index(content, ssMarkerStart)
		end := strings.Index(content, ssMarkerEnd) + len(ssMarkerEnd)
		content = strings.TrimRight(content[:start], "\r\n") + "\r\n" +
			strings.TrimLeft(content[end:], "\r\n")
	}

	if !enabled {
		return writeHosts(content)
	}

	var block strings.Builder
	block.WriteString("\r\n" + ssMarkerStart + "\r\n")
	for _, e := range safeSearchEntries {
		block.WriteString(e[0] + " " + e[1] + "\r\n")
	}
	block.WriteString(ssMarkerEnd + "\r\n")

	newContent := strings.TrimRight(content, "\r\n") + block.String()
	return writeHosts(newContent)
}

// IsSafeSearchActive returns true if the SafeSearch hosts entries are present.
func IsSafeSearchActive() bool {
	content, err := readHosts()
	if err != nil {
		return false
	}
	return strings.Contains(content, ssMarkerStart)
}

func readHosts() (string, error) {
	data, err := os.ReadFile(hostsPath())
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
	if err := os.WriteFile(hostsPath(), []byte(content), 0644); err == nil {
		flushDNS()
		return nil
	}
	return writeWithElevation(content)
}

// writeWithElevation writes a .ps1 script to %TEMP% and runs it elevated.
func writeWithElevation(content string) error {
	tmp := filepath.Join(os.TempDir(), "k10hosts_temp.txt")
	script := filepath.Join(os.TempDir(), "k10hosts_install.ps1")

	if err := os.WriteFile(tmp, []byte(content), 0666); err != nil {
		return fmt.Errorf("could not write temp hosts file: %w", err)
	}
	defer os.Remove(tmp)

	escapedTmp := strings.ReplaceAll(tmp, "'", "''")
	escapedDst := strings.ReplaceAll(hostsPath(), "'", "''")
	ps := "Copy-Item -Path '" + escapedTmp + "' -Destination '" + escapedDst + "' -Force\r\nipconfig /flushdns\r\n"

	if err := os.WriteFile(script, []byte(ps), 0666); err != nil {
		return fmt.Errorf("could not write script file: %w", err)
	}
	defer os.Remove(script)

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
