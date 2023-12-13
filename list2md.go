package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Repo describes a Github repository with additional field, last commit date
type Repo struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	DefaultBranch  string    `json:"default_branch"`
	Stars          int       `json:"stargazers_count"`
	Forks          int       `json:"forks_count"`
	Issues         int       `json:"open_issues_count"`
	Created        time.Time `json:"created_at"`
	Updated        time.Time `json:"updated_at"`
	URL            string    `json:"html_url"`
	LastCommitDate time.Time `json:"-"`
}

// HeadCommit describes a head commit of default branch
type HeadCommit struct {
	Sha    string `json:"sha"`
	Commit struct {
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

const (
	head = `# Top AI projects
A list of popular github projects related to AI (ranked by stars automatically)
Please update **list.txt** (via Pull Request)

<a href="./README.md">全部</a> |   <a href="./READMEpicture.md">图像</a> |   <a href="./READMEaudio.md">音频</a> | <a href="./READMEvideo.md">视频</a> | <a href="./READMElearn.md">学习</a> | 

| Project Name | Stars | Forks | Open Issues | Description | Last Commit |
| ------------ | ----- | ----- | ----------- | ----------- | ----------- |
`
	tail = "\n*Last Automatic Update: %v*"

	warning = "⚠️ No longer maintained ⚠️  "
)

var (
	deprecatedRepos = [3]string{"https://github.com/go-martini/martini", "https://github.com/pilu/traffic", "https://github.com/gorilla/mux"}
)

func main() {
	var wait sync.WaitGroup
	wait.Add(3)
	go func() {
		if err := generate(""); err != nil {
			fmt.Println("err generate main readme", err)
		}
		wait.Done()
	}()
	go func() {
		if err := generate("learn"); err != nil {
			fmt.Println("err generate learn readme", err)
		}

		wait.Done()
	}()
	go func() {
		if err := generate("picture"); err != nil {
			fmt.Println("err generate picture readme", err)
		}
		wait.Done()
	}()

	go func() {
		if err := generate("audio"); err != nil {
			fmt.Println("err generate audio readme", err)
		}
		wait.Done()
	}()
	wait.Wait()
}

func generate(category string) error {
	var repos []Repo
	accessToken := getAccessToken()

	byteContents, err := ioutil.ReadFile("list" + category + ".txt")
	if err != nil {
		return err
	}

	lines := strings.Split(string(byteContents), "\n")
	for _, url := range lines {
		if strings.HasPrefix(url, "https://github.com/") {
			var repo Repo
			var commit HeadCommit

			repoAPI := fmt.Sprintf(
				"https://api.github.com/repos/%s",
				strings.TrimFunc(url[19:], trimSpaceAndSlash),
			)
			fmt.Println(repoAPI)

			req, err := http.NewRequest(http.MethodGet, repoAPI, nil)
			if err != nil {
				return err
			}
			req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				log.Fatal(resp.Status)
			}

			decoder := json.NewDecoder(resp.Body)
			if err = decoder.Decode(&repo); err != nil {
				return err
			}

			commitAPI := fmt.Sprintf(
				"https://api.github.com/repos/%s/commits/%s",
				strings.TrimFunc(url[19:], trimSpaceAndSlash),
				repo.DefaultBranch,
			)
			fmt.Println(commitAPI)

			req, err = http.NewRequest(http.MethodGet, commitAPI, nil)
			if err != nil {
				return err
			}
			req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				log.Fatal(resp.Status)
			}

			decoder = json.NewDecoder(resp.Body)
			if err = decoder.Decode(&commit); err != nil {
				return err
			}

			repo.LastCommitDate = commit.Commit.Committer.Date
			repos = append(repos, repo)

			fmt.Printf("Repository: %v\n", repo)
			fmt.Printf("Head Commit: %v\n", commit)

			time.Sleep(3 * time.Second)
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Stars > repos[j].Stars
	})
	saveRanking(repos, category)
	return nil
}

func trimSpaceAndSlash(r rune) bool {
	return unicode.IsSpace(r) || (r == rune('/'))
}

func getAccessToken() string {
	tokenBytes, err := ioutil.ReadFile("access_token.txt")
	if err != nil {
		log.Fatal("Error occurs when getting access token")
	}
	return strings.TrimSpace(string(tokenBytes))
}

func saveRanking(repos []Repo, filesufix string) {
	readme, err := os.OpenFile("README"+filesufix+".md", os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer readme.Close()
	readme.WriteString(head)
	for _, repo := range repos {
		if isDeprecated(repo.URL) {
			repo.Description = warning + repo.Description
		}
		readme.WriteString(fmt.Sprintf("| [%s](%s) | %d | %d | %d | %s | %v |\n", repo.Name, repo.URL, repo.Stars, repo.Forks, repo.Issues, repo.Description, repo.LastCommitDate.Format("2006-01-02")))
	}
	readme.WriteString(fmt.Sprintf(tail, time.Now().Format(time.RFC3339)))
	readme.WriteString(`欢迎加入我们的社群 ![](https://raw.githubusercontent.com/mouuii/picture/master/weichat.jpg) `)
}

func isDeprecated(repoURL string) bool {
	for _, deprecatedRepo := range deprecatedRepos {
		if repoURL == deprecatedRepo {
			return true
		}
	}
	return false
}
