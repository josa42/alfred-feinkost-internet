package helper

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
)

type Options struct {
	Repo       string
	UpdateIcon string
	Run        func(wf *aw.Workflow, args []string)
}

type Package struct {
	updateJobName string
	updateIcon    *aw.Icon
	flagCheck     *bool
	flagLink      *bool
	flagRelease   *bool
	options       []aw.Option
	repo          string
	run           func(wf *aw.Workflow, args []string)
}

func New(o Options) Package {

	opts := []aw.Option{}
	if o.Repo != "" {
		opts = append(opts, update.GitHub(o.Repo))
	}

	var flagCheck bool
	flag.BoolVar(&flagCheck, "check", false, "Check for a new version")

	var flagLink bool
	flag.BoolVar(&flagLink, "link", false, "")

	var flagRelease bool
	flag.BoolVar(&flagRelease, "release", false, "")

	updateIcon := aw.IconInfo
	if o.UpdateIcon != "" {
		updateIcon = &aw.Icon{Value: o.UpdateIcon}
	}

	return Package{
		options:       opts,
		repo:          o.Repo,
		updateJobName: fmt.Sprintf("checkForUpdate.%x", md5.Sum([]byte(o.Repo))),
		updateIcon:    updateIcon,
		run:           o.Run,
		flagCheck:     &flagCheck,
		flagLink:      &flagLink,
		flagRelease:   &flagRelease,
	}
}

func (p *Package) Execute() {

	flag.Parse()

	if *p.flagLink {
		p.Link()
	} else if *p.flagRelease {
		p.Release()
	} else {
		p.Run()
	}
}

func (p *Package) Run() {
	wf := aw.New(p.options...)

	wf.Run(func() {
		args := wf.Args()
		flag.Parse()

		if wf.Updater != nil {
			if *p.flagCheck {
				p.runCheck(wf)
				return
			}
			p.runTriggerCheck(wf)

			if wf.UpdateAvailable() {
				wf.Configure(aw.SuppressUIDs(true))
				wf.NewItem("Update available!").
					Subtitle("↩ to install").
					Autocomplete("workflow:update").
					Valid(false).
					Icon(p.updateIcon)
			}
		}

		p.run(wf, args)
	})
}

func (p *Package) runCheck(wf *aw.Workflow) {
	wf.Configure(aw.TextErrors(true))
	log.Println("Checking for updates...")
	if err := wf.CheckForUpdate(); err != nil {
		wf.FatalError(err)
	}
}

func (p *Package) runTriggerCheck(wf *aw.Workflow) {
	if wf.UpdateCheckDue() && !wf.IsRunning(p.updateJobName) {
		log.Println("Running update check in background...")

		cmd := exec.Command(os.Args[0], "-check")
		if err := wf.RunInBackground(p.updateJobName, cmd); err != nil {
			log.Printf("Error starting update check: %s", err)
		}
	}
}

func (p *Package) Link() {
	pwd, _ := os.Getwd()

	usr, _ := user.Current()
	wfPath := filepath.Join(usr.HomeDir, "Library", "Application Support", "Alfred", "Alfred.alfredpreferences", "workflows", filepath.Base(pwd))

	fmt.Println(pwd)
	fmt.Println("Link: " + pwd + " -> " + wfPath)
	os.Symlink(pwd, wfPath)
}

func (p *Package) Release() {
	args := os.Args[2:]
	if len(args) != 1 {
		fmt.Println("Usage: ...")
		os.Exit(1)
	}

	name := regexp.MustCompile(`^[^/]+/`).ReplaceAllString(p.repo, "")

	pwd, _ := os.Getwd()

	version := args[0]

	must(os.RemoveAll("dist"))
	must(os.Mkdir("dist", 0777))

	sh("go", "build")

	sh("defaults", "write", filepath.Join(pwd, "info.plist"), "version", version)
	sh("plutil", "-convert", "xml1", filepath.Join(pwd, "info.plist"))
	sh("git", "add", "info.plist")

	replaceVersion("README.md", version)
	sh("git", "add", "README.md")

	// sh("git", "commit", "-m", fmt.Sprintf("🎉  Release %s", version))
	// sh("git", "push")

	sh("zip", "-r", "dist/"+name+"-"+version+".alfredworkflow", ".", "-x", "vendor*", ".git*", "bin*", "go.mod", "go.sum", "dist*", "README.md", "glide.lock", "*.go", "*.DS_Store", "docs/*")

	// sh("git", "tag", "version")
	// sh("git", "push", "origin", "refs/tags/"+version)
	//
	// sh(
	// 	"hub", "release", "create",
	// 	"-m", "🎉  Release "+version,
	// 	"-a", "dist/"+name+"-"+version+".alfredworkflow",
	// 	version,
	// )
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func sh(name string, args ...string) {

	fmt.Printf("> %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)

	out, err := cmd.CombinedOutput()

	if len(out) > 0 {
		fmt.Println(string(out))
	}
	must(err)
}

func replaceVersion(fpath, version string) {
	content, err := ioutil.ReadFile(fpath)
	must(err)

	re := regexp.MustCompile(`download/([^/]+)/(.+).alfredworkflow`)

	m := re.FindStringSubmatch(string(content))
	if len(m) > 0 {
		readme := strings.ReplaceAll(string(content), m[1], version)
		ioutil.WriteFile(fpath, []byte(readme), 0)
	}
}
