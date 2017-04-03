package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deiwin/gonfigure"
)

var (
	portProperty        = gonfigure.NewEnvProperty("PORT", "80")
	accessTokenProperty = gonfigure.NewRequiredEnvProperty("GITHUB_ACCESS_TOKEN")
	secretProperty      = gonfigure.NewRequiredEnvProperty("GITHUB_SECRET")
	// In the format defined in time.ParseDuration. E.g. "300ms", "-1.5h" or "2h45m".
	githubAPIDelayProperty = gonfigure.NewEnvProperty("GITHUB_API_DELAY", "2s")
	// A comma separated list of durations in the format defined in
	// time.ParseDuration. E.g. "300ms,1.5h,2h45m".
	githubAPITriesProperty = gonfigure.NewEnvProperty("GITHUB_API_TRIES", "0s,10s,30s,3m")
)

type Config struct {
	Port               int
	AccessToken        string
	Secret             string
	GithubAPIDelay     time.Duration
	GithubAPITryDeltas []time.Duration
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
		Port:               port,
		AccessToken:        accessTokenProperty.Value(),
		Secret:             secretProperty.Value(),
		GithubAPIDelay:     githubAPIDelay,
		GithubAPITryDeltas: getDeltasFromDurationsString(githubAPITriesProperty.Value()),
	}
}

func getDeltasFromDurationsString(durationsString string) []time.Duration {
	durationStringList := strings.Split(durationsString, ",")
	durationList := make([]time.Duration, len(durationStringList))
	for i, durationString := range durationStringList {
		var err error
		durationList[i], err = time.ParseDuration(durationString)
		if err != nil {
			panic(err)
		}
	}
	sort.Slice(durationList, func(i, j int) bool { return durationList[i] < durationList[j] })

	deltas := make([]time.Duration, len(durationList))
	for i, duration := range durationList {
		if i == 0 {
			deltas[i] = duration
		} else {
			deltas[i] = duration - durationList[i-1]
		}
	}
	return deltas
}
