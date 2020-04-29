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
	"github.com/palantir/go-githubapp/oauth2"
	"google.golang.org/appengine/log"
	"net/http"
	"os"

	"github.com/google/go-github/v30/github"
	"github.com/gregjones/httpcache"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rs/zerolog"
	"goji.io/pat"
)

func main() {
	config, err := ReadConfig("./config.yml")
	if err != nil {
		panic(err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	server, err := baseapp.NewServer(
		config.Server,
		baseapp.DefaultParams(logger, "exampleapp.")...,
	)
	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(
		config.Github,
		githubapp.WithClientUserAgent("example-app/1.0.0"),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
		githubapp.WithClientMiddleware(
			githubapp.ClientMetrics(server.Registry()),
		),
	)
	if err != nil {
		panic(err)
	}

	registerOAuth2Handler(config.Github)

	checkRunHandler := &StatusHandler{
		ClientCreator: cc,
		preamble:      config.AppConfig.PullRequestPreamble,
	}

	checkSuiteHandler := &CheckSuiteHandler{
		ClientCreator: cc,
		preamble:      config.AppConfig.PullRequestPreamble,
	}

	//webhookHandler := githubapp.NewDefaultEventDispatcher(config.Github, checkRunHandler)
	webhookHandler := githubapp.NewEventDispatcher(
		[]githubapp.EventHandler{
			checkRunHandler,
			checkSuiteHandler,
		},
		config.Github.App.WebhookSecret,
	)

	//server.Mux().Handle(pat.Post(githubapp.DefaultWebhookRoute), webhookHandler,webhookHandler2 )

	mux := server.Mux()

	// webhook route
	mux.Handle(pat.Post(githubapp.DefaultWebhookRoute), webhookHandler)

	// Start is blocking
	err = server.Start()
	if err != nil {
		panic(err)
	}
}

func registerOAuth2Handler(c githubapp.Config) {
	http.Handle("/api/auth/github", oauth2.NewHandler(
		oauth2.GetConfig(c, []string{"user:email"}),
		// force generated URLs to use HTTPS; useful if the app is behind a reverse proxy
		oauth2.ForceTLS(true),
		// set the callback for successful logins
		oauth2.OnLogin(func(w http.ResponseWriter, r *http.Request, login *oauth2.Login) {
			// look up the current user with the authenticated client
			client := github.NewClient(login.Client)
			user, _, _ := client.Users.Get(r.Context(), "")
			// handle error, save the user, ...
			log.Infof(r.Context(),"%v",user)

			// redirect the user back to another page
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		}),
	))
}

