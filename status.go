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
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type CheckRunHandler struct {
	githubapp.ClientCreator

	preamble string
}

func (h *CheckRunHandler) Handles() []string {
	return []string{"check_run"}
}

func (h *CheckRunHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.CheckRunEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment event payload")
	}

	zerolog.Ctx(ctx).Printf("%v, %v",event.GetAction(), event.GetCheckRun().GetApp().GetName())


	if event.GetAction() == "completed" {
		repo := event.GetRepo()

		prNum := event.GetCheckRun().PullRequests[0].GetNumber()

		installationID := githubapp.GetInstallationIDFromEvent(&event)

		ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, prNum)

		logger.Debug().Msgf("Event action is %s", event.GetAction())

		client, err := h.NewInstallationClient(installationID)
		if err != nil {
			return err
		}

		repoOwner := repo.GetOwner().GetLogin()
		repoName := repo.GetName()

		msg := fmt.Sprint(`
| Table | Markdown |
|-------|----------|
| works | yup     |
`)
		// yeah, IssueComment.
		// PRComments are review comments and have extra metadata
		prComment := github.IssueComment{
			Body: &msg,
		}

		if _, _, err := client.Issues.CreateComment(ctx, repoOwner, repoName, prNum, &prComment); err != nil {
			logger.Error().Err(err).Msg("Failed to comment on pull request")
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
