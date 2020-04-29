// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/jszwedko/go-circleci"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type StatusHandler struct {
	githubapp.ClientCreator

	preamble string
}

func (h *StatusHandler) Handles() []string {
	return []string{"status"}
}

func (h *StatusHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.StatusEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment event payload")
	}

	installationID := githubapp.GetInstallationIDFromEvent(&event)
	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	repo := event.GetRepo()
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	//https://circleci.com/gh/philipstaffordwood/hello-go-githubapp/24?utm_campaign=vcs-integration-link&utm_medium=referral&utm_source=github-build-link
	tUrl := event.GetTargetURL()
	tUrl = strings.Replace(tUrl, "https://circleci.com/gh/", "", 1)
	p := strings.Split(tUrl, "?")
	parts := strings.Split(p[0], "/")
	account := parts[0]
	circleRepo := parts[1]
	build, _ := strconv.Atoi(parts[2])

	circle := &circleci.Client{} // Token not required to query info for public projects

	//builds, _ := client.ListRecentBuildsForProject("jszwedko", "circleci-cli", "master", "", -1, 0)

	tests, _ := circle.ListTestMetadata(account, circleRepo, build)

	resultTable := `
|      | Message | Result |
|------|---------|--------|
`
	var failures bool = false
	for _, test := range tests {
		var resultSymbol string
		switch test.Result {
		case "success": resultSymbol = ":heavy_check_mark:" // green ✔️ on github
		case "failure": resultSymbol = ":x:"				// red ❌ on github
		case "skipped": resultSymbol = ":large_blue_circle:"// blue circle on github
		default: 		resultSymbol = test.Result
		}
		if test.Result != "success" {
			resultTable += fmt.Sprintf("| %v | `%v` | %v |\n", test.Classname, test.Name, resultSymbol)
			failures = true
		}}

	if failures {

		ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, repo)

		prs, err := ListOpenPullRequestsForSHA(ctx, client, repoOwner, repoName, event.GetSHA())
		if err != nil {
			return errors.Wrap(err, "failed to determine open pull requests matching the status context change")
		}

		for _, pr := range prs {
			msg := resultTable
			// yeah, IssueComment.
			// PRComments are review comments and have extra metadata
			prComment := github.IssueComment{
				Body: &msg,
			}

			if _, _, err := client.Issues.CreateComment(ctx, repoOwner, repoName, pr.GetNumber(), &prComment); err != nil {
				logger.Error().Err(err).Msg("Failed to comment on pull request")
			}
		}
	}

	//prNum := event.GetCheckRun().GetName()

	//ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, event.GetIssue().GetNumber())

	//logger.Debug().Msgf("Event action is %s", event.GetAction())
	//if event.GetAction() != "created" {
	//	return nil
	//}

	//repoOwner := repo.GetOwner().GetLogin()
	//repoName := repo.GetName()
	//author := event.GetComment().GetUser().GetLogin()
	//body := event.GetComment().GetBody()

	//if strings.HasSuffix(author, "[bot]") {
	//	logger.Debug().Msg("Issue comment was created by a bot")
	//	return nil
	//}

	//logger.Debug().Msgf("Creatinh comment on %s/%s#%d by %s", repoOwner, repoName, prNum, author)
	//msg := fmt.Sprintf("%s\n%s said\n```\n%s\n```\n", h.preamble, author, body)
	//prComment := github.IssueComment{
	//	Body: &msg,
	//}

	//if _, _, err := client.Issues.CreateComment(ctx, repoOwner, repoName, prNum, &prComment); err != nil {
	//	logger.Error().Err(err).Msg("Failed to comment on pull request")
	//}

	return nil
}

//FROM:
//https://github.com/palantir/bulldozer/blob/9f4be402a61ad47b3e0536c1b54839de9c4bc798/pull/pull_requests.go#L26-L92
// ListOpenPullRequestsForSHA returns all pull requests where the HEAD of the source branch
// in the pull request matches the given SHA.
func ListOpenPullRequestsForSHA(ctx context.Context, client *github.Client, owner, repoName, SHA string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

	openPRs, err := ListOpenPullRequests(ctx, client, owner, repoName)

	if err != nil {
		return nil, err
	}

	for _, openPR := range openPRs {
		if openPR.Head.GetSHA() == SHA {
			results = append(results, openPR)
		}
	}

	return results, nil
}

func ListOpenPullRequests(ctx context.Context, client *github.Client, owner, repoName string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

	opts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repoName, opts)
		if err != nil {
			return results, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repoName)
		}
		for _, pr := range prs {
			results = append(results, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return results, nil
}
