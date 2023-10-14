package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type GitHubActions struct {
	Client *github.Client
	Token  string
}

func NewGitHubActions(token string) *GitHubActions {
	ctx := context.Background()
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

	body, _ := ioutil.ReadAll(resp.Body)
	var response UploadUrlResponse
	json.Unmarshal(body, &response)

	// Upload artifact
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

func IsPullRequest() bool {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	headRef := os.Getenv("GITHUB_HEAD_REF")
	baseRef := os.Getenv("GITHUB_BASE_REF")

	return eventName == "pull_request" && headRef != "" && baseRef != ""
}
