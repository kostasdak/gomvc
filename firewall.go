// Firewall detection and notification
package gomvc

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// FirewallInfo struct contains information about detected firewall
type FirewallInfo struct {
	Detected    bool
	Name        string
	IsActive    bool
	Suggestions []string
}

// CheckFirewall checks for firewall and provides port opening suggestions
//
// This function detects the operating system and checks if a firewall is installed.
// If a firewall is detected, it provides helpful messages about opening the required port.
//
// Parameters:
//   - port: The port number your application is using
//
// Returns:
//   - FirewallInfo: Information about detected firewall and suggestions
func CheckFirewall(port int) FirewallInfo {
	info := FirewallInfo{
		Detected:    false,
		Suggestions: make([]string, 0),
	}

	os := runtime.GOOS

	switch os {
	case "linux":
		return checkLinuxFirewall(port)
	case "windows":
		return checkWindowsFirewall(port)
	case "darwin":
		return checkMacOSFirewall(port)
	default:
		InfoMessage(fmt.Sprintf("Operating System: %s (firewall check not supported)", os))
		return info
	}
}

// checkLinuxFirewall checks for Linux firewalls (ufw, firewalld, iptables)
func checkLinuxFirewall(port int) FirewallInfo {
	info := FirewallInfo{
		Detected:    false,
		Suggestions: make([]string, 0),
	}

	InfoMessage("Checking for firewall on Linux...")

	// Check for UFW (Ubuntu/Debian)
	if checkUFW(&info, port) {
		return info
	}

	// Check for firewalld (CentOS/RHEL/Fedora)
	if checkFirewalld(&info, port) {
		return info
	}

	// Check for iptables
	if checkIptables(&info, port) {
		return info
	}

	// No firewall detected
	InfoMessage("No active firewall detected")
	return info
}

// checkUFW checks for UFW firewall
func checkUFW(info *FirewallInfo, port int) bool {
	// Check if ufw command exists
	_, err := exec.LookPath("ufw")
	if err != nil {
		return false
	}

	info.Detected = true
	info.Name = "UFW (Uncomplicated Firewall)"

	// Check if UFW is active
	cmd := exec.Command("ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		InfoMessage("UFW detected but unable to check status")
		return true
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Status: active") {
		info.IsActive = true

		// Check if port is already allowed
		portStr := fmt.Sprintf("%d", port)
		if strings.Contains(outputStr, portStr) {
			InfoMessage(fmt.Sprintf("✓ UFW is active and port %d appears to be allowed", port))
		} else {
			InfoMessage(fmt.Sprintf("⚠ UFW firewall is ACTIVE - Port %d may be blocked!", port))
			InfoMessage(fmt.Sprintf("To allow port %d, run these commands:", port))
			InfoMessage(fmt.Sprintf("  sudo ufw allow %d/tcp", port))
			InfoMessage("  sudo ufw reload")

			info.Suggestions = append(info.Suggestions,
				fmt.Sprintf("sudo ufw allow %d/tcp", port),
				"sudo ufw reload",
			)
		}
	} else {
		InfoMessage("UFW detected but not active")
	}

	return true
}

// checkFirewalld checks for firewalld
func checkFirewalld(info *FirewallInfo, port int) bool {
	// Check if firewall-cmd exists
	_, err := exec.LookPath("firewall-cmd")
	if err != nil {
		return false
	}

	info.Detected = true
	info.Name = "firewalld"

	// Check if firewalld is running
	cmd := exec.Command("firewall-cmd", "--state")
	output, err := cmd.Output()
	if err != nil {
		InfoMessage("firewalld detected but not running")
		return true
	}

	if strings.TrimSpace(string(output)) == "running" {
		info.IsActive = true

		// Check if port is already allowed
		cmd = exec.Command("firewall-cmd", "--list-ports")
		output, err = cmd.Output()
		portStr := fmt.Sprintf("%d/tcp", port)

		if err == nil && strings.Contains(string(output), portStr) {
			InfoMessage(fmt.Sprintf("✓ firewalld is active and port %d appears to be allowed", port))
		} else {
			InfoMessage(fmt.Sprintf("⚠ firewalld is ACTIVE - Port %d may be blocked!", port))
			InfoMessage(fmt.Sprintf("To allow port %d, run these commands:", port))
			InfoMessage(fmt.Sprintf("  sudo firewall-cmd --permanent --add-port=%d/tcp", port))
			InfoMessage("  sudo firewall-cmd --reload")

			info.Suggestions = append(info.Suggestions,
				fmt.Sprintf("sudo firewall-cmd --permanent --add-port=%d/tcp", port),
				"sudo firewall-cmd --reload",
			)
		}
	} else {
		InfoMessage("firewalld detected but not running")
	}

	return true
}

// checkIptables checks for iptables
func checkIptables(info *FirewallInfo, port int) bool {
	// Check if iptables exists
	_, err := exec.LookPath("iptables")
	if err != nil {
		return false
	}

	info.Detected = true
	info.Name = "iptables"

	// Check if there are any iptables rules
	cmd := exec.Command("iptables", "-L", "-n")
	output, err := cmd.Output()
	if err != nil {
		InfoMessage("iptables detected but unable to check rules (may need sudo)")
		return true
	}

	outputStr := string(output)
	// Check if there are actual rules (not just default chains)
	if strings.Contains(outputStr, "Chain INPUT") && len(outputStr) > 200 {
		info.IsActive = true
		InfoMessage(fmt.Sprintf("⚠ iptables firewall detected - Port %d may be blocked!", port))
		InfoMessage(fmt.Sprintf("To allow port %d with iptables, run:", port))
		InfoMessage(fmt.Sprintf("  sudo iptables -A INPUT -p tcp --dport %d -j ACCEPT", port))
		InfoMessage("  sudo iptables-save | sudo tee /etc/iptables/rules.v4")

		info.Suggestions = append(info.Suggestions,
			fmt.Sprintf("sudo iptables -A INPUT -p tcp --dport %d -j ACCEPT", port),
			"sudo iptables-save | sudo tee /etc/iptables/rules.v4",
		)
	} else {
		InfoMessage("iptables detected but appears to have no active rules")
	}

	return true
}

// checkWindowsFirewall checks for Windows Firewall
func checkWindowsFirewall(port int) FirewallInfo {
	info := FirewallInfo{
		Detected:    true,
		Name:        "Windows Defender Firewall",
		Suggestions: make([]string, 0),
	}

	InfoMessage("Checking Windows Firewall...")

	// Try to check Windows Firewall status
	cmd := exec.Command("netsh", "advfirewall", "show", "allprofiles", "state")
	output, err := cmd.Output()

	if err != nil {
		InfoMessage("Windows Firewall detected but unable to check status")
		InfoMessage(fmt.Sprintf("⚠ Port %d may be blocked by Windows Firewall", port))
		InfoMessage("To allow port in Windows Firewall, run PowerShell as Administrator:")
		InfoMessage(fmt.Sprintf("  New-NetFirewallRule -DisplayName 'GoMVC App' -Direction Inbound -LocalPort %d -Protocol TCP -Action Allow", port))

		info.Suggestions = append(info.Suggestions,
			fmt.Sprintf("New-NetFirewallRule -DisplayName 'GoMVC App' -Direction Inbound -LocalPort %d -Protocol TCP -Action Allow", port),
		)
		return info
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "ON") {
		info.IsActive = true
		InfoMessage("⚠ Windows Firewall is ACTIVE")
		InfoMessage(fmt.Sprintf("Port %d may be blocked by Windows Firewall", port))
		InfoMessage("To allow port, run in PowerShell as Administrator:")
		InfoMessage(fmt.Sprintf("  New-NetFirewallRule -DisplayName 'GoMVC App' -Direction Inbound -LocalPort %d -Protocol TCP -Action Allow", port))

		info.Suggestions = append(info.Suggestions,
			fmt.Sprintf("New-NetFirewallRule -DisplayName 'GoMVC App' -Direction Inbound -LocalPort %d -Protocol TCP -Action Allow", port),
		)
	} else {
		InfoMessage("Windows Firewall appears to be disabled")
	}

	return info
}

// checkMacOSFirewall checks for macOS firewall
func checkMacOSFirewall(port int) FirewallInfo {
	info := FirewallInfo{
		Detected:    true,
		Name:        "macOS Application Firewall",
		Suggestions: make([]string, 0),
	}

	InfoMessage("Checking macOS Firewall...")

	// Check if firewall is enabled
	cmd := exec.Command("defaults", "read", "/Library/Preferences/com.apple.alf", "globalstate")
	output, err := cmd.Output()

	if err != nil {
		InfoMessage("Unable to check macOS firewall status")
		return info
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "1" || outputStr == "2" {
		info.IsActive = true
		InfoMessage("⚠ macOS Firewall is ACTIVE")
		InfoMessage(fmt.Sprintf("Port %d may be blocked", port))
		InfoMessage("To allow your application:")
		InfoMessage("  1. Go to System Preferences > Security & Privacy > Firewall")
		InfoMessage("  2. Click 'Firewall Options'")
		InfoMessage("  3. Add your application to the allowed list")

		info.Suggestions = append(info.Suggestions,
			"Add application in System Preferences > Security & Privacy > Firewall",
		)
	} else {
		InfoMessage("macOS Firewall appears to be disabled")
	}

	return info
}

// DisplayFirewallHelp displays helpful firewall information
func DisplayFirewallHelp(port int) {

	InfoMessage(CenterText("FIREWALL CONFIGURATION CHECK", 40, '='))
	firewallInfo := CheckFirewall(port)

	if firewallInfo.Detected {
		InfoMessage(fmt.Sprintf("Firewall: %s", firewallInfo.Name))
		InfoMessage(fmt.Sprintf("Status: %s", getStatusText(firewallInfo.IsActive)))

		if firewallInfo.IsActive && len(firewallInfo.Suggestions) > 0 {
			InfoMessage("")
			InfoMessage("Suggested commands to open port:")
			for _, suggestion := range firewallInfo.Suggestions {
				InfoMessage("  " + suggestion)
			}
		}
	} else {
		InfoMessage("No firewall detected or firewall is not active")
	}
}

// getStatusText returns human-readable status
func getStatusText(active bool) string {
	if active {
		return "ACTIVE"
	}
	return "Inactive"
}
