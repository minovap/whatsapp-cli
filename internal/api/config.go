package api

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIKey         string
	Port           int
	StoreDir       string
	MaxMessages    int
	MaxHours       int
	PhoneWhitelist []string
	PhoneBlacklist []string
	LogLevel       string
}

func ParseConfig() (Config, error) {
	c := Config{
		APIKey:   os.Getenv("API_KEY"),
		Port:     8080,
		StoreDir: "/data/store",
		MaxMessages: 100,
		MaxHours:    48,
		LogLevel:    "info",
	}

	if c.APIKey == "" {
		return Config{}, fmt.Errorf("API_KEY environment variable is required")
	}

	if v := os.Getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT value: %s", v)
		}
		c.Port = port
	}

	if v := os.Getenv("STORE_DIR"); v != "" {
		c.StoreDir = v
	}

	if v := os.Getenv("MAX_MESSAGES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid MAX_MESSAGES value: %s", v)
		}
		c.MaxMessages = n
	}

	if v := os.Getenv("MAX_HOURS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid MAX_HOURS value: %s", v)
		}
		c.MaxHours = n
	}

	if v := os.Getenv("PHONE_WHITELIST"); v != "" {
		c.PhoneWhitelist = splitAndTrim(v)
	}

	if v := os.Getenv("PHONE_BLACKLIST"); v != "" {
		c.PhoneBlacklist = splitAndTrim(v)
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}

	return c, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
