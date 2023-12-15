package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func createOrSwitchBranch(branchName string) {
	if err := exec.Command("git", "rev-parse", "--verify", branchName).Run(); err != nil {
		err = exec.Command("git", "switch", "-c", branchName).Run()

		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := exec.Command("git", "switch", branchName).Run(); err != nil {
		log.Fatal(err)
	}
}

func createEmptyCommit() {
	output, err := exec.Command("git", "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		log.Fatal(err)
	}

	if len(output) == 0 {
		cmd := exec.Command("git", "commit", "--allow-empty", "-m", "[skip ci] REMOVE ME. EMPTY COMMIT", "--no-verify")
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func createPR(jiraID, jiraTitle string) {
	title := fmt.Sprintf("\"%s: %s\"", jiraID, jiraTitle)

	var body string
	// TODO: currently doesn't support multiple templates
	templatePath := filepath.Join(".github", "pull_request_template.md")
	templateContent, err := os.ReadFile(templatePath)
	if err == nil {
		body = "\"" + string(templateContent) + "\""
	} else {
		body = "\"\""
	}

	// workaround for determine the default push target branch
	if err := exec.Command("git", "push", "-u", "origin", "HEAD").Run(); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("gh", "pr", "create", "-d", "-t", title, "-b", body)
	fmt.Println(cmd)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	var jiraID string

	if len(os.Args) > 1 {
		jiraID = os.Args[1]
	} else {
		fmt.Scanln(&jiraID)
	}

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

	createOrSwitchBranch(branchName)
	createEmptyCommit()
	createPR(jiraID, jiraTitle)
}
