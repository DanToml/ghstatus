package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
)

const (
	BANNER = `       _         _        _
  __ _| |__  ___| |_ __ _| |_ _   _ ___
 / _` + "`" + ` | '_ \/ __| __/ _` + "`" + ` | __| | | / __|
| (_| | | | \__ \ || (_| | |_| |_| \__ \
 \__, |_| |_|___/\__\__,_|\__|\__,_|___/
 |___/

 Summarise the open issues and pull-requests on your GitHub Repositories.`
)

var (
	token string
	orgs  stringSlice

	debug bool
)

func init() {
	flag.StringVar(&token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	flag.Var(&orgs, "orgs", "organizations to include")
	flag.BoolVar(&debug, "d", false, "run in debug mode")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, BANNER)
		flag.PrintDefaults()
	}

	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if token == "" {
		fmt.Fprint(os.Stderr, "A GitHub token is required.\n\n")
		flag.Usage()
		fmt.Fprint(os.Stderr, "\n")
		os.Exit(1)
	}
}

func makeGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return client
}

func main() {
	ctx := context.Background()
	client := makeGitHubClient(ctx, token)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		logrus.Fatal(err)
	}
	username := *user.Login

	orgs = append(orgs, username)

	page := 1
	perPage := 100
	logrus.Debugf("Getting repositories...")
	if err := handleRepositories(ctx, client, page, perPage); err != nil {
		logrus.Fatal(err)
	}
}

func handleRepositories(ctx context.Context, client *github.Client, page, perPage int) error {
	opt := &github.RepositoryListOptions{
		Affiliation: "owner,collaborator,organization_member",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := client.Repositories.List(ctx, "", opt)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		logrus.Debugf("Handling repo %s...", *repo.FullName)
		if err := handleRepo(ctx, client, repo); err != nil {
			logrus.Warn(err)
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return handleRepositories(ctx, client, page, perPage)
}

func handleRepo(ctx context.Context, client *github.Client, repo *github.Repository) error {
	if !shouldHandleRepo(repo) {
		return nil
	}

	logger := logrus.WithFields(logrus.Fields{"owner": *repo.Owner.Login, "name": *repo.Name})
	if *repo.OpenIssuesCount == 0 {
		logger.Debug("No open issues")
	} else {
		logger.Infof("Open Issues Count: %d", *repo.OpenIssuesCount)
	}
	return nil
}

func shouldHandleRepo(repo *github.Repository) bool {
	for _, org := range orgs {
		if org == *repo.Owner.Login {
			return true
		}
	}
	return false
}

// Typealias for slice of strings to allow flag support.

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%s", *s)
}
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
