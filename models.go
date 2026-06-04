package main

// Config holds the application configuration loaded from config.yaml
type Config struct {
	SimklClientID string            `yaml:"simkl_client_id"`
	AppName       string            `yaml:"app_name"`
	ShokoToken    string            `yaml:"shoko_token"`
	ShokoURL      string            `yaml:"shoko_url"`
	NtfyURL       string            `yaml:"ntfy_url"`
	NtfyToken     string            `yaml:"ntfy_token"`
	NtfyTopic     string            `yaml:"ntfy_topic"`
	Users         map[string]string `yaml:"users"`
}
