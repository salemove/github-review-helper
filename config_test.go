package main_test

import (
	"fmt"
	"os"
	"time"

	grh "github.com/salemove/github-review-helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type envVar struct {
	name  string
	value string
}

var _ = Describe("Config", func() {
	Describe("GITHUB_ACCESS_TOKEN", func() {
		name := "GITHUB_ACCESS_TOKEN"

		Context("when set", func() {
			token := "my-github-token"
			setEnvVars(replaceEnvVarByName(name, token, requiredEnvVars))

			It("is passed as a string", func() {
				conf := grh.NewConfig()
				Expect(conf.AccessToken).To(Equal(token))
			})
		})

		Context("when not set", func() {
			setEnvVars(omitEnvVarByName(name, requiredEnvVars))

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})
	})

	Describe("GITHUB_SECRET", func() {
		name := "GITHUB_SECRET"

		Context("when set", func() {
			secret := "my-github-secret"
			setEnvVars(replaceEnvVarByName(name, secret, requiredEnvVars))

			It("is passed as a string", func() {
				conf := grh.NewConfig()
				Expect(conf.Secret).To(Equal(secret))
			})
		})

		Context("when not set", func() {
			setEnvVars(omitEnvVarByName(name, requiredEnvVars))

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})
	})

	Describe("PORT", func() {
		name := "PORT"

		Context("when set", func() {
			port := "1337"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: port})

			It("is passed as an int", func() {
				conf := grh.NewConfig()
				Expect(conf.Port).To(Equal(1337))
			})
		})

		Context("when not a number", func() {
			port := "leet"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: port})

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})

		Context("when not set", func() {
			setEnvVars(requiredEnvVars)

			It("defaults to a value", func() {
				conf := grh.NewConfig()
				Expect(conf.Port).NotTo(BeNil())
			})
		})
	})

	Describe("GITHUB_API_TRIES", func() {
		name := "GITHUB_API_TRIES"

		Context("when set to sorted durations", func() {
			durationsString := "0s,2m,2h,4h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("is passed as an array of duration deltas", func() {
				conf := grh.NewConfig()
				Expect(conf.GithubAPITryDeltas).To(Equal([]time.Duration{
					0,
					2 * time.Minute,
					time.Hour + 58*time.Minute,
					2 * time.Hour,
				}))
			})
		})

		Context("when set to unsorted durations", func() {
			durationsString := "0s,2h,2m,4h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("is passed as a sorted array of duration deltas", func() {
				conf := grh.NewConfig()
				Expect(conf.GithubAPITryDeltas).To(Equal([]time.Duration{
					0,
					2 * time.Minute,
					time.Hour + 58*time.Minute,
					2 * time.Hour,
				}))
			})
		})

		Context("when contains an invalid duration", func() {
			durationsString := "two minutes,2h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})

		Context("when contains an empty duration", func() {
			durationsString := ",2h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})

		Context("when contains spaces", func() {
			durationsString := "2m, 2h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})

		Context("when contains negative durations", func() {
			durationsString := "2m,2h,-2h"
			setEnvVars(requiredEnvVars)
			setEnvVar(envVar{name: name, value: durationsString})

			It("panics", func() {
				Expect(func() {
					grh.NewConfig()
				}).To(Panic())
			})
		})

		Context("when not set", func() {
			setEnvVars(requiredEnvVars)

			It("defaults to a value", func() {
				conf := grh.NewConfig()
				Expect(conf.GithubAPITryDeltas).NotTo(BeNil())
			})
		})
	})
})

var setEnvVar = func(variable envVar) {
	var (
		previousValue       string
		previousValueWasSet bool
	)

	BeforeEach(func() {
		previousValue, previousValueWasSet = os.LookupEnv(variable.name)
		err := os.Setenv(variable.name, variable.value)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		if previousValueWasSet {
			err := os.Setenv(variable.name, previousValue)
			Expect(err).NotTo(HaveOccurred())
		} else {
			err := os.Unsetenv(variable.name)
			Expect(err).NotTo(HaveOccurred())
		}
	})
}

var setEnvVars = func(vars []envVar) {
	for _, variable := range vars {
		setEnvVar(variable)
	}
}

var requiredEnvVars = []envVar{
	{name: "GITHUB_ACCESS_TOKEN", value: "token"},
	{name: "GITHUB_SECRET", value: "secret"},
}

var replaceEnvVarByName = func(nameToReplace, newValue string, vars []envVar) []envVar {
	newVars := make([]envVar, len(vars))
	indexReplaced := -1
	for i, variable := range vars {
		if variable.name == nameToReplace {
			newVars[i] = envVar{name: nameToReplace, value: newValue}
			indexReplaced = i
		} else {
			newVars[i] = variable
		}
	}
	if indexReplaced == -1 {
		panic(fmt.Sprintf("Couldn't find env var %s in %v", nameToReplace, vars))
	}

	return newVars
}

// omitEnvVarByName "omits" env variables from the list by setting their value
// to an empty string. This will look as if the value has not been set (or has
// been set to an empty string) even if the value exists in the env outside of
// this test run.
var omitEnvVarByName = func(nameToOmit string, vars []envVar) []envVar {
	return replaceEnvVarByName(nameToOmit, "", vars)
}
