package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type JiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

func getJiraIssues(jiraAPIToken, jiraDomain string) []JiraIssue {
	url := fmt.Sprintf("https://%s/rest/api/2/search?jql=status='To Do'", jiraDomain)
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

	var jiraIssues []JiraIssue
	err = json.Unmarshal(body, &jiraIssues)
	if err != nil {
		log.Fatal(err)
	}

	return jiraIssues
}

func selectJiraIssue(issues []JiraIssue) string {
	var keys []string
	for _, issue := range issues {
		keys = append(keys, fmt.Sprintf("%s - %s", issue.Key, issue.Fields.Summary))
	}

	cmd := exec.Command("fzf", "--height", "50%", "--prompt", "Select a Jira issue: ")
	cmd.Stdin = strings.NewReader(strings.Join(keys, "\n"))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running fzf: %v\n%s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String())
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

	jiraIssues := getJiraIssues(jiraAPIToken, jiraDomain)
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
