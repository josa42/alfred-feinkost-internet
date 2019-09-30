package main

// Package is called aw
import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
)

const (
	updateJobName = "checkForUpdate"
	repo          = "josa42/alfred-feinkost-internet"
)

var (
	flagCheck     bool
	wf            *aw.Workflow
	iconAvailable = &aw.Icon{Value: "icon/update.png"}
)

func init() {
	wf = aw.New(update.GitHub(repo))

	flag.BoolVar(&flagCheck, "check", false, "Check for a new version")
}

func main() {
	wf.Run(run)
}

type index struct {
	Title    string    `json:"title"`
	Episodes []episode `json:"episodes"`
}

type episode struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	FileURL    string   `json:"file_url"`
	RecordedAt string   `json:"recorded_at"`
	Duration   string   `json:"duration"`
	Team       []string `json:"team"`
}

func run() {
	args := wf.Args()
	flag.Parse()

	query := strings.Join(args, " ")

	if flagCheck {
		runCheck()
		return
	}

	runTriggerCheck()

	wf.Configure(aw.SuppressUIDs(true))

	if query == "" && wf.UpdateAvailable() {
		wf.NewItem("Update available!").
			Subtitle("↩ to install").
			Autocomplete("workflow:update").
			Valid(false).
			Icon(iconAvailable)
	}

	resp, _ := http.Get("https://feinkost-internet.de/index.json")
	body, _ := ioutil.ReadAll(resp.Body)

	index := index{}
	json.Unmarshal(body, &index)

	for _, e := range index.Episodes {

		if strings.Contains(strings.ToLower(e.Title), query) {
			wf.
				NewItem(e.Title).
				Subtitle("Open in Browser").
				Arg(e.URL).
				Valid(true).
				NewModifier("alt").
				Subtitle("Play in QuickTime Player").
				Arg(e.FileURL)
		}

	}

	wf.SendFeedback()
}

func runCheck() {
	wf.Configure(aw.TextErrors(true))
	log.Println("Checking for updates...")
	if err := wf.CheckForUpdate(); err != nil {
		wf.FatalError(err)
	}
}

func runTriggerCheck() {
	if wf.UpdateCheckDue() && !wf.IsRunning(updateJobName) {
		log.Println("Running update check in background...")

		cmd := exec.Command(os.Args[0], "-check")
		if err := wf.RunInBackground(updateJobName, cmd); err != nil {
			log.Printf("Error starting update check: %s", err)
		}
	}
}
