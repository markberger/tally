package tally

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type GithubRepo struct {
	repo         string
	interval     time.Duration
	pullRequests map[int]pullRequest
	bot          *Bot
}

type pullRequest struct {
	Url        string
	Number     int
	State      string
	Title      string
	Created_at string
	Updated_at string
}

func (bot *Bot) NewGithubRepo(repo string, interval time.Duration) *GithubRepo {
	g := new(GithubRepo)
	g.repo = repo
	g.bot = bot
	g.pullRequests = g.getPullRequests()
	return g
}

func (g *GithubRepo) parsePullRequests(jsonBlob []byte) []pullRequest {
	var requests []pullRequest
	err := json.Unmarshal(jsonBlob, &requests)
	if err != nil {
		log.Printf("Cannot unmarshal JSON blob with pull requests for %s:\n", g.repo)
		log.Printf("%v\n", err)
	}
	return requests
}

func (g *GithubRepo) fetchPullRequests(state string) []byte {
	url := "https://api.github.com/repos/" + g.repo + "/pulls?state=" + state
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Unable to fetch pull requests for %s:\n", g.repo)
		log.Printf("%v\n", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading JSON body for %s:\n", url)
		log.Printf("%v\n", err)
	}
	return body
}

func (g *GithubRepo) getPullRequests() map[int]pullRequest {
	jsonBlob := g.fetchPullRequests("open")
	openPRs := g.parsePullRequests(jsonBlob)
	jsonBlob = g.fetchPullRequests("closed")
	closedPRs := g.parsePullRequests(jsonBlob)
	m := make(map[int]pullRequest)
	pullRequests := append(openPRs, closedPRs...)
	for i := range pullRequests {
		m[pullRequests[i].Number] = pullRequests[i]
	}
	return m
}

func (g *GithubRepo) CheckPullRequests() {
	pullRequests := g.getPullRequests()
	for prNum, pr := range pullRequests {
		_, ok := g.pullRequests[prNum]
		if !ok {
			pullRequests[prNum] = pr
		}

		if pr.Created_at == pr.Updated_at {
			msg := "New pull request #" + strconv.Itoa(prNum) + ": " + pr.Title
			g.bot.MsgChannel(msg)
			g.bot.MsgChannel(pr.Url)
		} else if g.pullRequests[prNum].State != pr.State {
			// If a pull request has been opened or closed
			msg := "Pull request \"" + pr.Title + "\"  is now " + pr.State + "."
			g.bot.MsgChannel(msg)
			g.bot.MsgChannel(pr.Url)
			g.pullRequests[prNum] = pr
		} else if g.pullRequests[prNum].Updated_at != pr.Updated_at {
			// If a pull request has been updated
			msg := "Pull request \"" + pr.Title + "\" has been updated."
			g.bot.MsgChannel(msg)
			g.bot.MsgChannel(pr.Url)
			g.pullRequests[prNum] = pr
		}
	}
}

func (g *GithubRepo) Run() {
	for {
		g.CheckPullRequests()
		time.Sleep(g.interval * time.Second)
	}
}
