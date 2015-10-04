package main

import (
	"strconv"

	"github.com/deiwin/gonfigure"
)

var (
	portProperty        = gonfigure.NewEnvProperty("PORT", "80")
	accessTokenProperty = gonfigure.NewRequiredEnvProperty("GITHUB_ACCESS_TOKEN")
	secretProperty      = gonfigure.NewRequiredEnvProperty("GITHUB_SECRET")
)

type Config struct {
	Port        int
	AccessToken string
	Secret      string
}

func NewConfig() Config {
	port, err := strconv.Atoi(portProperty.Value())
	if err != nil {
		panic(err)
	}
	return Config{
		Port:        port,
		AccessToken: accessTokenProperty.Value(),
		Secret:      secretProperty.Value(),
	}
}
