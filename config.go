package main

import (
	"fmt"
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
	// A comma separated list of durations in the format defined in
	// time.ParseDuration. E.g. "300ms,1.5h,2h45m". When first duration is 0,
	// then GitHub API requests will initially be tried synchronously and only
	// the retries will be asynchronous.
	githubAPITriesProperty = gonfigure.NewEnvProperty("GITHUB_API_TRIES", "0s,10s,30s,3m")
)

type Config struct {
	Port               int
	AccessToken        string
	Secret             string
	GithubAPITryDeltas []time.Duration
}

func NewConfig() Config {
	port, err := strconv.Atoi(portProperty.Value())
	if err != nil {
		panic(err)
	}

	githubAPITryDeltas, err := getDeltasFromDurationsString(githubAPITriesProperty.Value())
	if err != nil {
		panic(fmt.Sprintf("Failed to get deltas from GITHUB_API_TRIES durations string: %v", err))
	}

	return Config{
		Port:               port,
		AccessToken:        accessTokenProperty.Value(),
		Secret:             secretProperty.Value(),
		GithubAPITryDeltas: githubAPITryDeltas,
	}
}

func getDeltasFromDurationsString(durationsString string) ([]time.Duration, error) {
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
	// After sorting, the first element will be the smallest, so checking the
	// first value for negative durations is enough.
	if durationList[0] < 0 {
		return nil, fmt.Errorf("Unable to parse \"%s\" - negative durations are not supported.", durationsString)
	}

	deltas := make([]time.Duration, len(durationList))
	for i, duration := range durationList {
		if i == 0 {
			deltas[i] = duration
		} else {
			deltas[i] = duration - durationList[i-1]
		}
	}
	return deltas, nil
}
