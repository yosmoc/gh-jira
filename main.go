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
	"regexp"
	"strings"
)

var EMPTY_COMMIT_MESSAGE = "[skip ci] REMOVE ME. EMPTY COMMIT"

type Component struct {
	Name string `json:"name"`
}

type JiraIssue struct {
	IssueID string `json:"key"`
	Fields  struct {
		Summary    string      `json:"summary"`
		Components []Component `json:"components"`
	} `json:"fields"`
}

type JiraResponse struct {
	Issues []JiraIssue `json:"issues"`
}

func getJiraIssues(jiraAPIToken, jiraDomain, bID, bStatus string) []JiraIssue {
	jql := fmt.Sprintf("project=%s AND status=%s", bID, bStatus)
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

	var jiraResponse JiraResponse
	err = json.Unmarshal(body, &jiraResponse)
	if err != nil {
		log.Fatal(err)
	}

	return jiraResponse.Issues
}

func selectJiraIssue(issues []JiraIssue, iFilter string) (string, string) {
	keys := make(map[string]JiraIssue)
	var fzfInput []string
	for _, issue := range issues {
		for _, component := range issue.Fields.Components {
			if component.Name == iFilter {
				key := fmt.Sprintf("%s - %s", issue.IssueID, issue.Fields.Summary)
				keys[key] = issue
				fzfInput = append(fzfInput, key)
				break
			}
		}
	}

	cmd := exec.Command("fzf", "--height", "50%", "--prompt", "Select a Jira issue: ")
	cmd.Stdin = strings.NewReader(strings.Join(fzfInput, "\n"))
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error running fzf: %v", err)
	}

	selectedKey := strings.TrimSpace(string(output))
	selectedIssue := keys[selectedKey]

	return selectedIssue.IssueID, selectedIssue.Fields.Summary

	// Debug
	// fmt.Printf("Selected issue: %s\n", string(output))
	// fmt.Printf("Jira issue id: %s\n", strings.SplitN(string(output), " ", 2)[0])
	// fmt.Printf("Jira issue title: %s\n", strings.Join(strings.SplitN(string(output), " ", 10)[2:], "-"))
}

func createBranch(issueID, issueTitle string) {
	sanitizedIssueTitle := strings.ReplaceAll(issueTitle, " ", "-")
	reg, err := regexp.Compile("[^a-zA-Z0-9-]+")
	if err != nil {
		log.Fatal(err)
	}
	sanitizedIssueTitle = reg.ReplaceAllString(sanitizedIssueTitle, "")
	sanitizedIssueTitle = strings.ToLower(sanitizedIssueTitle)

	branchName := fmt.Sprintf("%s/%s", issueID, sanitizedIssueTitle)

	if err := exec.Command("git", "rev-parse", "--verify", branchName).Run(); err != nil {
		err = exec.Command("git", "switch", "-c", branchName).Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if err := exec.Command("git", "switch", branchName).Run(); err != nil {
			log.Fatal(err)
		}
	}

	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	if len(output) == 0 {
		cmd = exec.Command("git", "commit", "--allow-empty", "-m", EMPTY_COMMIT_MESSAGE, "--no-verify")
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

	var boardID = "UTPR"                     // Utsikt - Presence Current Sprint.
	var boardStatus = "'To Do'"              // Issue Status, must be in single quotes.
	var issueFilter = "Software Engineering" // Issue Filter.

	jiraIssues := getJiraIssues(jiraAPIToken, jiraDomain, boardID, boardStatus)
	if len(jiraIssues) == 0 {
		log.Fatal("No Jira issues with 'To Do' status found.")
	}

	issueID, issueSummary := selectJiraIssue(jiraIssues, issueFilter)
	if issueID == "" || issueSummary == "" {
		log.Fatal("No Jira issue selected. Manual entry is required.")
	}

	fmt.Printf("Selected issue: %s - %s", issueID, issueSummary)

	createBranch(issueID, issueSummary)
	// // Extract Jira ID and Title from the selected issue
	// parts := strings.SplitN(selectedIssue, " - ", 2)
	// if len(parts) != 2 {
	// 	log.Fatal("Error parsing selected Jira issue.")
	// }
	// jiraID, jiraTitle := parts[0], parts[1]

	// createPR(jiraID, jiraTitle)
}
