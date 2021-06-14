package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

type Status struct {
	Class string `json:"_class"`
	Links struct {
		PrevRun struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"prevRun"`
		Parent struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"parent"`
		Tests struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"tests"`
		Nodes struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"nodes"`
		Log struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"log"`
		Self struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"self"`
		BlueTestSummary struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"blueTestSummary"`
		Actions struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"actions"`
		Steps struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"steps"`
		ChangeSet struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"changeSet"`
		Artifacts struct {
			Class string `json:"_class"`
			Href  string `json:"href"`
		} `json:"artifacts"`
	} `json:"_links"`
	Actions          []interface{} `json:"actions"`
	ArtifactsZipFile interface{}   `json:"artifactsZipFile"`
	CauseOfBlockage  interface{}   `json:"causeOfBlockage"`
	Causes           []struct {
		Class            string `json:"_class"`
		ShortDescription string `json:"shortDescription"`
	} `json:"causes"`
	Description               string      `json:"description"`
	DurationInMillis          int         `json:"durationInMillis"`
	EnQueueTime               string      `json:"enQueueTime"`
	EndTime                   string      `json:"endTime"`
	EstimatedDurationInMillis int         `json:"estimatedDurationInMillis"`
	ID                        string      `json:"id"`
	Name                      interface{} `json:"name"`
	Organization              string      `json:"organization"`
	Pipeline                  string      `json:"pipeline"`
	Replayable                bool        `json:"replayable"`
	Result                    string      `json:"result"`
	RunSummary                string      `json:"runSummary"`
	StartTime                 string      `json:"startTime"`
	State                     string      `json:"state"`
	Type                      string      `json:"type"`
	ChangeSet                 []struct {
		Class string `json:"_class"`
		Links struct {
			Self struct {
				Class string `json:"_class"`
				Href  string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
		AffectedPaths []string `json:"affectedPaths"`
		Author        struct {
			Class string `json:"_class"`
			Links struct {
				Favorites struct {
					Class string `json:"_class"`
					Href  string `json:"href"`
				} `json:"favorites"`
				Self struct {
					Class string `json:"_class"`
					Href  string `json:"href"`
				} `json:"self"`
			} `json:"_links"`
			Avatar     interface{} `json:"avatar"`
			Email      interface{} `json:"email"`
			FullName   string      `json:"fullName"`
			ID         string      `json:"id"`
			Permission interface{} `json:"permission"`
		} `json:"author"`
		CheckoutCount int           `json:"checkoutCount"`
		CommitID      string        `json:"commitId"`
		Issues        []interface{} `json:"issues"`
		Msg           string        `json:"msg"`
		Timestamp     string        `json:"timestamp"`
		URL           interface{}   `json:"url"`
	} `json:"changeSet"`
	Branch struct {
		IsPrimary bool          `json:"isPrimary"`
		Issues    []interface{} `json:"issues"`
		URL       string        `json:"url"`
	} `json:"branch"`
	CommitID    string      `json:"commitId"`
	CommitURL   interface{} `json:"commitUrl"`
	PullRequest interface{} `json:"pullRequest"`
}

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		log.Fatal("must provide -h host (i.e. https://servername)")
	}
	host := args[1]
	uri := "blue/rest/organizations/jenkins/pipelines/CN2/runs/"
	url := fmt.Sprintf("%s/%s", host, uri)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	jsonResp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	var statusList []Status
	if err := json.Unmarshal(jsonResp, &statusList); err != nil {
		log.Fatal(err)
	}

	var errorMap2 = make(map[string]map[string]CommitURL)
	for _, status := range statusList {
		if status.Result == "FAILURE" && status.Branch.URL == "master" {
			logFileURL := fmt.Sprintf("%s/%s", host, status.Links.Log.Href)
			logContent, err := getLog(logFileURL)
			if err != nil {
				log.Fatal(err)
			}
			errors := extractError(logContent)

			for _, e := range errors {
				e = strings.TrimRight(e, "\r")
				if e == "script returned exit code 2" {
					continue
				}
				commitURL := CommitURL{
					Commit: status.CommitID,
					URL:    logFileURL,
				}
				if _, ok := errorMap2[e]; !ok {
					commitURLMap := map[string]CommitURL{status.ID: commitURL}
					errorMap2[e] = commitURLMap
				} else {
					if _, ok := errorMap2[e][status.ID]; !ok {
						errorMap2[e][status.ID] = commitURL
					}
				}
			}
		}
	}

	rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
	t := table.NewWriter()
	t.AppendHeader(table.Row{"ERROR TYPE", "RUN", "URL"}, rowConfigAutoMerge)
	for k, v := range errorMap2 {
		for k2, v2 := range v {
			t.AppendRow(table.Row{k, k2, v2.URL}, rowConfigAutoMerge)
		}
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true},
	})
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = true
	t.SortBy([]table.SortBy{
		{Name: "RUN", Mode: table.Dsc},
	})
	/*
		t.SetColumnConfigs([]table.ColumnConfig{{
			Name:     "URL",
			WidthMin: 6,
			WidthMax: 64,
		}},
		)
	*/
	fmt.Println(t.Render())

}

func jsonPrettyPrint(in []byte) error {
	var out bytes.Buffer
	err := json.Indent(&out, in, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(out.String())
	return nil
}

func getLog(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	logResp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(logResp), nil
}

func extractError(text string) []string {
	re := regexp.MustCompile(`\[.*\] (?i)ERROR: (.*)`)
	found := re.FindAllStringSubmatch(text, -1)
	var errorList []string
	for _, f := range found {
		if len(f) > 0 {
			errorList = append(errorList, f[1])
		}
	}
	return errorList
}

type ErrorMsg struct {
	Commit   string
	Messages []string
}

type CommitURL struct {
	Commit string
	URL    string
}
