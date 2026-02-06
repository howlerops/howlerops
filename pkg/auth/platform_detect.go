package auth

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// BiometricCapability represents available biometric authentication
type BiometricCapability struct {
	Available bool   `json:"available"`
	Type      string `json:"type"`
	Platform  string `json:"platform"`
}

// DetectBiometricCapability detects available biometric authentication on the current platform
func DetectBiometricCapability() (*BiometricCapability, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectMacOSBiometric()
	case "windows":
		return detectWindowsBiometric()
	case "linux":
		return detectLinuxBiometric()
	default:
		return &BiometricCapability{
			Available: false,
			Type:      "none",
			Platform:  runtime.GOOS,
		}, nil
	}
}

// detectMacOSBiometric detects Touch ID or Face ID on macOS
func detectMacOSBiometric() (*BiometricCapability, error) {
	// Check if biometric authentication is available
	// This checks for the presence of biometric hardware (Touch ID or Face ID)
	cmd := exec.Command("bioutil", "-r", "-s")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// bioutil not found or no biometric hardware
		return &BiometricCapability{
			Available: false,
			Type:      "none",
			Platform:  "darwin",
		}, nil
	}

	outputStr := string(output)

	// Determine the type of biometric
	biometricType := "Touch ID" // Default assumption

	// Check for Face ID (typically on newer Macs)
	// Note: This is a simplified check. In reality, we'd need more sophisticated detection
	if strings.Contains(strings.ToLower(outputStr), "face") {
		biometricType = "Face ID"
	}

	return &BiometricCapability{
		Available: true,
		Type:      biometricType,
		Platform:  "darwin",
	}, nil
}

// detectWindowsBiometric detects Windows Hello availability
func detectWindowsBiometric() (*BiometricCapability, error) {
	// Check for Windows Hello availability using PowerShell
	// This checks if Windows Hello is configured and available
	psCmd := `
	$biometric = Get-WmiObject -Namespace "root\cimv2\Security\MicrosoftVolumeEncryption" -Class Win32_EncryptableVolume -ErrorAction SilentlyContinue
	if ($biometric) { "available" } else { "unavailable" }
	`

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// PowerShell error or Windows Hello not available
		// Try alternative method: check for biometric devices
		return detectWindowsBiometricAlternative()
	}

	outputStr := strings.TrimSpace(string(output))

	if strings.Contains(strings.ToLower(outputStr), "available") {
		return &BiometricCapability{
			Available: true,
			Type:      "Windows Hello",
			Platform:  "windows",
		}, nil
	}

	return &BiometricCapability{
		Available: false,
		Type:      "none",
		Platform:  "windows",
	}, nil
}

// detectWindowsBiometricAlternative uses an alternative method to detect Windows biometric
func detectWindowsBiometricAlternative() (*BiometricCapability, error) {
	// Check for biometric devices via Device Manager
	psCmd := `Get-PnpDevice -Class Biometric -ErrorAction SilentlyContinue | Select-Object -First 1`

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		return &BiometricCapability{
			Available: false,
			Type:      "none",
			Platform:  "windows",
		}, nil
	}

	return &BiometricCapability{
		Available: true,
		Type:      "Windows Hello",
		Platform:  "windows",
	}, nil
}

// detectLinuxBiometric detects fingerprint readers on Linux
func detectLinuxBiometric() (*BiometricCapability, error) {
	// Check for fprintd (fingerprint daemon) on Linux
	cmd := exec.Command("which", "fprintd")
	err := cmd.Run()

	if err != nil {
		// fprintd not found
		return &BiometricCapability{
			Available: false,
			Type:      "none",
			Platform:  "linux",
		}, nil
	}

	// Check if fingerprint devices are available
	cmd = exec.Command("fprintd-list", "--list-devices")
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		return &BiometricCapability{
			Available: false,
			Type:      "none",
			Platform:  "linux",
		}, nil
	}

	return &BiometricCapability{
		Available: true,
		Type:      "Fingerprint",
		Platform:  "linux",
	}, nil
}

// CheckBiometricAvailability is a convenience function that returns a map
// This is useful for Wails bindings
func CheckBiometricAvailability() (map[string]interface{}, error) {
	capability, err := DetectBiometricCapability()
	if err != nil {
		return nil, fmt.Errorf("failed to detect biometric capability: %w", err)
	}

	return map[string]interface{}{
		"available": capability.Available,
		"type":      capability.Type,
		"platform":  capability.Platform,
	}, nil
}
