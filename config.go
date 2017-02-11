package main

import (
	"strconv"
	"time"

	"github.com/deiwin/gonfigure"
)

var (
	portProperty        = gonfigure.NewEnvProperty("PORT", "80")
	accessTokenProperty = gonfigure.NewRequiredEnvProperty("GITHUB_ACCESS_TOKEN")
	secretProperty      = gonfigure.NewRequiredEnvProperty("GITHUB_SECRET")
	// In the format defined in time.ParseDuration. E.g. "300ms", "-1.5h" or "2h45m".
	githubAPIDelayProperty = gonfigure.NewEnvProperty("GITHUB_API_DELAY", "2s")
)

type Config struct {
	Port           int
	AccessToken    string
	Secret         string
	GithubAPIDelay time.Duration
}

func NewConfig() Config {
	port, err := strconv.Atoi(portProperty.Value())
	if err != nil {
		panic(err)
	}

	githubAPIDelay, err := time.ParseDuration(githubAPIDelayProperty.Value())
	if err != nil {
		panic(err)
	}

	return Config{
		Port:           port,
		AccessToken:    accessTokenProperty.Value(),
		Secret:         secretProperty.Value(),
		GithubAPIDelay: githubAPIDelay,
	}
}
