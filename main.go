package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/josa42/alfred-feinkost-internet/helper"
)

var (
	nameExp  = regexp.MustCompile(`@([^\s]+)`)
	spaceExp = regexp.MustCompile(`\s+`)
)

type index struct {
	Title    string    `json:"title"`
	Episodes []episode `json:"episodes"`
}

type episode struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	FileURL    string `json:"file_url"`
	RecordedAt string `json:"recorded_at"`
	Duration   string `json:"duration"`
	Team       Team   `json:"team"`
}

type Team []string

func (t Team) ContainsAll(names []string) bool {
	for _, member := range t {
		for _, n := range names {
			if strings.ToLower(member) == n {
				return true
			}
		}
	}
	return false
}

func main() {
	pkg := helper.New(helper.Options{
		Repo:       "josa42/alfred-feinkost-internet",
		UpdateIcon: "icon/update.png",
		Run: func(wf *aw.Workflow, args []string) {
			query, names := parseArgs(args)

			wf.Configure(aw.SuppressUIDs(true))

			resp, _ := http.Get("https://feinkost-internet.de/index.json")
			body, _ := ioutil.ReadAll(resp.Body)

			index := index{}
			json.Unmarshal(body, &index)

			for _, e := range index.Episodes {
				if len(names) > 0 && !e.Team.ContainsAll(names) {
					continue
				}

				if strings.Contains(strings.ToLower(e.Title), query) {
					wf.
						NewItem(e.Title).
						Subtitle(fmt.Sprintf("Team: %s", strings.Join(e.Team, ", "))).
						Arg(e.URL).
						Valid(true).
						Quicklook(e.URL).
						NewModifier("alt").
						Subtitle("Play in QuickTime Player").
						Arg(e.FileURL)
				}
			}

			wf.SendFeedback()
		},
	})

	pkg.Execute()
}

func parseArgs(args []string) (string, []string) {
	query := strings.Join(args, " ")

	names := []string{}
	for _, name := range nameExp.FindAllString(query, -1) {
		names = append(names, nameExp.ReplaceAllString(name, "$1"))
	}

	query = nameExp.ReplaceAllString(query, " ")
	query = spaceExp.ReplaceAllString(query, " ")

	return strings.TrimSpace(query), names
}
