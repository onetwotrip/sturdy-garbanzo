package main

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/onetwotrip/go-bitbucket"
	"github.com/onetwotrip/gojenkins"
	"strings"
)

type config struct {
	BitbucketUser string   `env:"BITBUCKET_USER"`
	BitbucketPass string   `env:"BITBUCKET_PASS"`
	JenkinsUrl    string   `env:"JENKINS_URL"`
	JenkinsUser   string   `env:"JENKINS_USER"`
	JenkinsPass   string   `env:"JENKINS_PASS"`
	RepoOwner     string   `env:"REPO_OWNER"`
	SkipList      []string `env:"SKIP_LIST" envSeparator:","`
	ReferenceJob  string   `env:"REFERENCE_JOB"`
}

var (
	skip = map[string]bool{}
)

func main() {
	cfg := config{}

	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	for _, skipElement := range cfg.SkipList {
		skip[skipElement] = true
	}

	c := bitbucket.NewBasicAuth(cfg.BitbucketUser, cfg.BitbucketPass)
	j, err := gojenkins.CreateJenkins(nil, cfg.JenkinsUrl, cfg.JenkinsUser, cfg.JenkinsPass).Init()
	if err != nil {
		panic(err)
	}

	opt := &bitbucket.RepositoriesOptions{
		Owner: cfg.RepoOwner,
	}

	res, err := c.Repositories.ListForAccount(opt)
	if err != nil {
		panic(err)
	}

	a := res.(map[string]interface{})
	b := a["values"].([]interface{})

	for _, repository := range b {
		repoSlug := repository.(map[string]interface{})["slug"].(string)

		if skip[repoSlug] {
			fmt.Printf("will skip %s due config\n", repoSlug)
			continue
		}

		opt := &bitbucket.RepositoryOptions{
			Owner:     cfg.RepoOwner,
			Repo_slug: repoSlug,
			File:      "Jenkinsfile",
		}

		res, err = c.Repositories.Repository.GetFile(opt)
		if err == nil {
			_, err := j.GetJob(repoSlug)
			if err != nil {
				fmt.Printf("job %s does not exist, will create\n", repoSlug)
				job, _ := j.GetJob(cfg.ReferenceJob)
				jobCopy, err := job.Copy(repoSlug)
				if jobCopy.GetName() == repoSlug {
					fmt.Printf("created job for %s", repoSlug)
					config, _ := jobCopy.GetConfig()
					err = jobCopy.UpdateConfig(strings.Replace(config, cfg.ReferenceJob, repoSlug, 1))
					if err != nil {
						fmt.Printf("can't fix config for %s\n", repoSlug)
					}
					fmt.Printf("fixed confog for %s\n", repoSlug)
				} else {
					fmt.Printf("can't create job: %s", err)
				}
			}
		}
	}
}
