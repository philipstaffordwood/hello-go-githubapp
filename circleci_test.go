package main

import (
	"fmt"
	"testing"

	"github.com/jszwedko/go-circleci"
)

func Test(t *testing.T) {
	client := &circleci.Client{} // Token not required to query info for public projects

	//builds, _ := client.ListRecentBuildsForProject("jszwedko", "circleci-cli", "master", "", -1, 0)

	artifacts, _ := client.ListBuildArtifacts("flanksource","platform-cli",3040)

	for _, artifact := range artifacts {
		fmt.Printf("%#v", artifact)
	}
}
