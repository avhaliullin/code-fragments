package config

import (
	"fmt"
	"os"
)

type Config struct {
	Database         string
	DatabaseEndpoint string
}

func LoadFromEnv() *Config {
	return &Config{
		Database:         requireString("DATABASE"),
		DatabaseEndpoint: requireString("DATABASE_ENDPOINT"),
	}
}

func requireString(name string) string {
	res, ok := os.LookupEnv(name)
	if !ok {
		panic(fmt.Sprintf("required env var %s not found", name))
	}
	return res
}
