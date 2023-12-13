package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type JiraResponse struct {
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

func getJiraTitle(jiraID, jiraAPIToken, jiraDomain string) string {
	url := fmt.Sprintf("https://%s/rest/api/2/issue/%s", jiraDomain, jiraID)
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

	var jiraResp JiraResponse
	err = json.Unmarshal(body, &jiraResp)
	if err != nil {
		log.Fatal(err)
	}

	return jiraResp.Fields.Summary
}

func createBranch(branchName string) {
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

func createPR(jiraID, jiraTitle string) {
	cmd := exec.Command("gh", "pr", "create", "--title", fmt.Sprintf("%s: %s", jiraID, jiraTitle), "-w")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide a Jira ID")
	}
	jiraID := os.Args[1]

	jiraAPIToken := os.Getenv("JIRA_API_TOKEN")
	if jiraAPIToken == "" {
		log.Fatal("Please provide a Jira API Token")
	}

	jiraDomain := os.Getenv("JIRA_DOMAIN")
	if jiraDomain == "" {
		log.Fatal("Please provide a Jira Domain")
	}

	jiraTitle := getJiraTitle(jiraID, jiraAPIToken, jiraDomain)
	jiraTitleInBranchName := strings.ReplaceAll(jiraTitle, " ", "_")
	branchName := fmt.Sprintf("%s/%s", jiraID, jiraTitleInBranchName)

	createBranch(branchName)
	createPR(jiraID, jiraTitle)
}
