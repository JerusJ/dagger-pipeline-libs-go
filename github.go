package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

var (
	ErrNoOpenPullRequests = errors.New("ERROR: No open pull requests found for branch.")
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

func (gha *GitHubActions) CommentOrUpdatePR(ctx context.Context, owner string, repo string, prNumber int, newComment string, identifier string) error {
	// List comments on the PR
	comments, _, err := gha.Client.Issues.ListComments(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return err
	}

	// Check for an existing comment with the identifier
	for _, comment := range comments {
		if strings.Contains(comment.GetBody(), identifier) {
			// Update the existing comment
			comment.Body = &newComment
			_, _, err := gha.Client.Issues.EditComment(ctx, owner, repo, *comment.ID, comment)
			return err
		}
	}

	// Create a new comment if no existing comment is found
	_, _, err = gha.Client.Issues.CreateComment(ctx, owner, repo, prNumber, &github.IssueComment{Body: &newComment})
	return err
}

func (gha *GitHubActions) GetOpenPullRequestIDForBranch(ctx context.Context, owner string, repo string, branch string) (int, error) {
	opts := &github.PullRequestListOptions{
		State: "open",
	}

	pulls, _, err := gha.Client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return 0, err
	}

	for _, pull := range pulls {
		if *pull.Head.Ref == branch {
			return *pull.Number, nil
		}
	}

	return 0, ErrNoOpenPullRequests
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
