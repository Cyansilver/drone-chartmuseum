package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"code.gitea.io/git"
	"github.com/honestbee/drone-chartmuseum/pkg/cmclient"
	"github.com/honestbee/drone-chartmuseum/pkg/util"
	"k8s.io/helm/pkg/chartutil"
)

type (

	// Config struct map with drone plugin parameters
	Config struct {
		RepoURL          string `json:"repo_url,omitempty"`
		ChartPath        string `json:"chart_path,omitempty"`
		ChartDir         string `json:"chart_dir,omitempty"`
		SaveDir          string `json:"save_dir,omitempty"`
		PreviousCommitID string `json:"previous_commit_id,omitempty"`
		CurrentCommitID  string `json:"current_commit_id,omitempty"`
	}

	// Plugin struct
	Plugin struct {
		Config Config
	}
)

// GetDiffFiles : similar to git diff, get the file changes between 2 commits
func (p *Plugin) GetDiffFiles() []string {
	fmt.Printf("Getting diff between %v and %v ...\n", p.Config.PreviousCommitID, p.Config.CurrentCommitID)
	repository, err := git.OpenRepository(p.Config.ChartDir)
	if err != nil {
		log.Fatal(err)
	}

	commit, err := repository.GetCommit(p.Config.CurrentCommitID)
	if err != nil {
		log.Fatal(err)
	}

	files, err := commit.GetFilesChangedSinceCommit(p.Config.PreviousCommitID)
	if err != nil {
		log.Fatal(err)
	}

	return files
}

// SaveChartToPackage : save helm chart folder to compressed package
func (p *Plugin) SaveChartToPackage(chartPath string) (string, error) {
	var message string
	var err error
	if _, err := os.Stat(p.Config.SaveDir); os.IsNotExist(err) {
		os.Mkdir(p.Config.SaveDir, os.ModePerm)
	}

	if ok, _ := chartutil.IsChartDir(chartPath); ok == true {
		c, _ := chartutil.LoadDir(chartPath)
		message, err = chartutil.Save(c, p.Config.SaveDir)
		if err != nil {
			log.Printf("%v : %v", chartPath, err)
		}
		fmt.Printf("packaging %v ...\n", message)
	}

	return message, err
}

func (p *Plugin) defaultExec(files []string) {
	var resultList []string
	for _, file := range files {
		chart, err := p.SaveChartToPackage(file)
		if err == nil {
			resultList = append(resultList, chart)
		}
	}
	cmclient.UploadToServer(resultList, p.Config.RepoURL)
}

func (p *Plugin) exec() error {
	var files []string
	if p.Config.ChartPath != "" {
		files = []string{p.Config.ChartPath}
	} else if p.Config.PreviousCommitID != "" && p.Config.CurrentCommitID != "" {
		diffFiles := p.GetDiffFiles()
		files = util.GetParentFolders(util.FilterExtFiles(diffFiles))
	} else {
		dirs, err := ioutil.ReadDir(p.Config.ChartDir)
		if err != nil {
			log.Fatal(err)
		}
		files = util.ExtractDirs(dirs)
	}

	p.defaultExec(files)
	return nil
}
