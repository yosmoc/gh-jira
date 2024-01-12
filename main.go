package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Component struct {
	Name string `json:"name"`
}

type JiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary    string      `json:"summary"`
		Components []Component `json:"components"`
	} `json:"fields"`
}

type JiraResponse struct {
	Issues []JiraIssue `json:"issues"`
}

func getJiraIssues(jiraAPIToken, jiraDomain, boardID string) []JiraIssue {
	jql := fmt.Sprintf("project=%s AND status='To Do'", boardID)
	url := fmt.Sprintf("https://%s/rest/api/2/search?jql=%s", jiraDomain, url.QueryEscape(jql))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+jiraAPIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Received non-OK response code: %d\nResponse body: %s", resp.StatusCode, body)
	}

	var jiraResponse JiraResponse
	err = json.Unmarshal(body, &jiraResponse)
	if err != nil {
		log.Fatal(err)
	}

	return jiraResponse.Issues
}

func selectJiraIssue(issues []JiraIssue) string {
	var keys []string
	for _, issue := range issues {
		for _, component := range issue.Fields.Components {
			if component.Name == "Software Engineering" {
				keys = append(keys, fmt.Sprintf("%s - %s", issue.Key, issue.Fields.Summary))
				break
			}
		}
	}

	for _, key := range keys {
		fmt.Println(key)
	}

	// cmd := exec.Command("fzf", "--height", "50%", "--prompt", "Select a Jira issue: ")
	// cmd.Stdin = strings.NewReader(strings.Join(keys, "\n"))

	// var stdout, stderr bytes.Buffer
	// cmd.Stdout = &stdout
	// cmd.Stderr = &stderr

	// err := cmd.Run()
	// if err != nil {
	// 	log.Fatalf("Error running fzf: %v\n%s", err, stderr.String())
	// }

	// return strings.TrimSpace(stdout.String())
	return ""
}

func createBranch(jiraID, jiraTitle string) {
	branchName := fmt.Sprintf("%s/%s", jiraID, strings.ReplaceAll(jiraTitle, " ", "-"))
	exec.Command("git", "switch", "-c", branchName).Run()

	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	if len(output) == 0 {
		cmd = exec.Command("git", "commit", "--allow-empty", "-m", "[skip ci] REMOVE ME. EMPTY COMMIT", "--no-verify")
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// func createPR(jiraID, jiraTitle string) {
// 	cmd := exec.Command("gh", "pr", "create", "--title", fmt.Sprintf("%s: %s", jiraID, jiraTitle), "-w")
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	err := cmd.Run()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func main() {
	jiraAPIToken := os.Getenv("JIRA_API_TOKEN")
	if jiraAPIToken == "" {
		log.Fatal("Please provide a Jira API Token")
	}

	jiraDomain := os.Getenv("JIRA_DOMAIN")
	if jiraDomain == "" {
		log.Fatal("Please provide a Jira Domain")
	}

	boardID := "UTPR"

	jiraIssues := getJiraIssues(jiraAPIToken, jiraDomain, boardID)
	if len(jiraIssues) == 0 {
		log.Fatal("No Jira issues with 'To Do' status found.")
	}

	selectedIssue := selectJiraIssue(jiraIssues)
	if selectedIssue == "" {
		log.Fatal("No Jira issue selected. Manual entry is required.")
	}

	// Extract Jira ID and Title from the selected issue
	parts := strings.SplitN(selectedIssue, " - ", 2)
	if len(parts) != 2 {
		log.Fatal("Error parsing selected Jira issue.")
	}
	jiraID, jiraTitle := parts[0], parts[1]

	createBranch(jiraID, jiraTitle)
	// createPR(jiraID, jiraTitle)
}
