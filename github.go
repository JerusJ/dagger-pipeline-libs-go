package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type GitHubActions struct {
	Client *github.Client
	Token  string
}

func NewGitHubActions(ctx context.Context, token string) *GitHubActions {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GitHubActions{
		Client: client,
		Token:  token,
	}
}

type UploadUrlResponse struct {
	Url string `json:"upload_url"`
}

func (gha *GitHubActions) UploadArtifact(ctx context.Context, repo string, runID int64, artifactName string, artifactPath string) error {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/artifacts", repo, runID)

	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Set("Authorization", "Bearer "+gha.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response UploadUrlResponse
	json.Unmarshal(body, &response)

	zipData, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}

	uploadUrl := fmt.Sprintf("%s?name=%s", response.Url, artifactName)
	req, _ = http.NewRequest("POST", uploadUrl, bytes.NewBuffer(zipData))
	req.Header.Set("Authorization", "Bearer "+gha.Token)
	req.Header.Set("Content-Type", "application/zip")

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("Failed to upload artifact: %s", resp.Status)
	}
	return nil
}

func (gha *GitHubActions) DownloadArtifact(ctx context.Context, owner string, repo string, runID int64, artifactName string, destination string) error {
	artifacts, _, err := gha.Client.Actions.ListArtifacts(ctx, owner, repo, &github.ListOptions{})
	if err != nil {
		return err
	}

	var artifact *github.Artifact
	for _, a := range artifacts.Artifacts {
		if a.GetName() == artifactName {
			artifact = a
			break
		}
	}

	if artifact == nil {
		return fmt.Errorf("Artifact %s not found", artifactName)
	}
	_, resp, err := gha.Client.Actions.DownloadArtifact(ctx, owner, repo, *artifact.ID, 3)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func AddGithubOutputShell(name, value string) {
	cmd := fmt.Sprintf("::set-output name=%s::%s", name, value)
	fmt.Fprintln(os.Stdout, cmd)
}

func IsPullRequest() bool {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	headRef := os.Getenv("GITHUB_HEAD_REF")
	baseRef := os.Getenv("GITHUB_BASE_REF")

	return eventName == "pull_request" && headRef != "" && baseRef != ""
}
