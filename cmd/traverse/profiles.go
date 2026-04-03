package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Profile struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

type ProfilesConfig struct {
	Profiles       []Profile `json:"profiles"`
	DefaultProfile string    `json:"default_profile,omitempty"`
}

func getProfilesPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	traverseDir := filepath.Join(homeDir, ".traverse")
	if err := os.MkdirAll(traverseDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(traverseDir, "profiles.json"), nil
}

func loadProfiles() (*ProfilesConfig, error) {
	path, err := getProfilesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfilesConfig{
				Profiles: make([]Profile, 0),
			}, nil
		}
		return nil, err
	}

	var cfg ProfilesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func saveProfiles(cfg *ProfilesConfig) error {
	path, err := getProfilesPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func loadProfile(name string) (*Profile, error) {
	cfg, err := loadProfiles()
	if err != nil {
		return nil, err
	}

	// If no profiles exist and name matches a known profile, use default
	if len(cfg.Profiles) == 0 && name == "default" {
		return nil, fmt.Errorf("no default profile configured")
	}

	// Try to find the profile
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == name {
			return &cfg.Profiles[i], nil
		}
	}

	// If not found and name is "default", try to find default
	if name == "default" && cfg.DefaultProfile != "" {
		for i := range cfg.Profiles {
			if cfg.Profiles[i].Name == cfg.DefaultProfile {
				return &cfg.Profiles[i], nil
			}
		}
	}

	return nil, fmt.Errorf("profile '%s' not found", name)
}

func profileListCommand() {
	cfg, err := loadProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles saved yet.")
		fmt.Println("Use 'traverse profile create -name <name> -url <url>' to create a profile.")
		return
	}

	fmt.Printf("Saved Profiles (%d):\n\n", len(cfg.Profiles))

	for _, p := range cfg.Profiles {
		marker := ""
		if p.Name == cfg.DefaultProfile {
			marker = " [DEFAULT]"
		}
		fmt.Printf("  %s%s\n", p.Name, marker)
		fmt.Printf("    URL: %s\n", p.URL)
		if p.Username != "" {
			fmt.Printf("    User: %s\n", p.Username)
		}
		if p.Token != "" {
			fmt.Printf("    Auth: Bearer token\n")
		}
		fmt.Println()
	}
}

func profileCreateCommand(name, url, user, pass, token string) error {
	if name == "" {
		return fmt.Errorf("profile name is required (-name)")
	}
	if url == "" {
		return fmt.Errorf("URL is required (-url)")
	}

	cfg, err := loadProfiles()
	if err != nil {
		return err
	}

	// Check if profile already exists
	for _, p := range cfg.Profiles {
		if p.Name == name {
			return fmt.Errorf("profile '%s' already exists", name)
		}
	}

	profile := Profile{
		Name:     name,
		URL:      url,
		Username: user,
		Password: pass,
		Token:    token,
	}

	cfg.Profiles = append(cfg.Profiles, profile)

	if err := saveProfiles(cfg); err != nil {
		return err
	}

	fmt.Printf("Profile '%s' created successfully.\n", name)
	return nil
}

func profileDeleteCommand(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is required")
	}

	cfg, err := loadProfiles()
	if err != nil {
		return err
	}

	found := false
	newProfiles := make([]Profile, 0)

	for _, p := range cfg.Profiles {
		if p.Name != name {
			newProfiles = append(newProfiles, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("profile '%s' not found", name)
	}

	cfg.Profiles = newProfiles

	// If the deleted profile was the default, clear it
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}

	if err := saveProfiles(cfg); err != nil {
		return err
	}

	fmt.Printf("Profile '%s' deleted successfully.\n", name)
	return nil
}

func profileSetDefaultCommand(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is required")
	}

	cfg, err := loadProfiles()
	if err != nil {
		return err
	}

	found := false
	for _, p := range cfg.Profiles {
		if p.Name == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("profile '%s' not found", name)
	}

	cfg.DefaultProfile = name

	if err := saveProfiles(cfg); err != nil {
		return err
	}

	fmt.Printf("Profile '%s' set as default.\n", name)
	return nil
}
