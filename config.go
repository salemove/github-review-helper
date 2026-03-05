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
	portProperty              = gonfigure.NewEnvProperty("PORT", "80")
	accessTokenProperty       = gonfigure.NewEnvProperty("GITHUB_ACCESS_TOKEN", "")
	secretProperty            = gonfigure.NewRequiredEnvProperty("GITHUB_SECRET")
	appIDProperty             = gonfigure.NewEnvProperty("GITHUB_APP_ID", "")
	appPrivateKeyFileProperty = gonfigure.NewEnvProperty("GITHUB_APP_PRIVATE_KEY_FILE", "")
	appInstallationIDProperty = gonfigure.NewEnvProperty("GITHUB_APP_INSTALLATION_ID", "")
	// A comma separated list of durations in the format defined in
	// time.ParseDuration. E.g. "300ms,1.5h,2h45m". When first duration is 0,
	// then GitHub API requests will initially be tried synchronously and only
	// the retries will be asynchronous.
	githubAPITriesProperty = gonfigure.NewEnvProperty("GITHUB_API_TRIES", "0s,10s,30s,3m")
)

type Config struct {
	Port               int
	AccessToken        string
	AppID              int64
	AppPrivateKeyFile  string
	AppInstallationID  int64
	Secret             string
	GithubAPITryDeltas []time.Duration
}

func (c Config) IsAppAuth() bool {
	return c.AppID != 0
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

	accessToken := accessTokenProperty.Value()
	appIDStr := appIDProperty.Value()
	appPrivateKeyFile := appPrivateKeyFileProperty.Value()
	appInstallationIDStr := appInstallationIDProperty.Value()

	hasPatAuth := accessToken != ""
	hasAppAuth := appIDStr != "" || appPrivateKeyFile != "" || appInstallationIDStr != ""

	if hasPatAuth && hasAppAuth {
		panic("Cannot configure both PAT (GITHUB_ACCESS_TOKEN) and GitHub App authentication. Choose one.")
	}
	if !hasPatAuth && !hasAppAuth {
		panic("Must configure either GITHUB_ACCESS_TOKEN or all three GITHUB_APP_* variables.")
	}

	var appID, appInstallationID int64
	if hasAppAuth {
		if appIDStr == "" || appPrivateKeyFile == "" || appInstallationIDStr == "" {
			panic("GitHub App auth requires all three: GITHUB_APP_ID, GITHUB_APP_PRIVATE_KEY_FILE, GITHUB_APP_INSTALLATION_ID")
		}
		appID, err = strconv.ParseInt(appIDStr, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("GITHUB_APP_ID must be a number: %v", err))
		}
		appInstallationID, err = strconv.ParseInt(appInstallationIDStr, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("GITHUB_APP_INSTALLATION_ID must be a number: %v", err))
		}
	}

	return Config{
		Port:               port,
		AccessToken:        accessToken,
		AppID:              appID,
		AppPrivateKeyFile:  appPrivateKeyFile,
		AppInstallationID:  appInstallationID,
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
