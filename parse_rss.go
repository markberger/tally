package tally

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"time"
	"net/http"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel
}

type Channel struct {
	XMLName xml.Name `xml:"channel"`
	Title   string   `xml:"title"`
	Link    string   `xml:"link"`
	Items   []Item   `xml:"item"`
}

type Item struct {
	XMLName xml.Name `xml:"item"`
	Title   string   `xml:"title"`
	Author  string   `xml:"creator"`
	Link    string   `xml:"link"`
}

type TimelineUpdater struct {
	last_item Item
	feed      string
	interval  time.Duration
	bot       *Bot
}

func (bot *Bot) NewTimelineUpdater(feed string, interval time.Duration) *TimelineUpdater {
	t := new(TimelineUpdater)
	t.feed = feed
	t.interval = interval
	t.bot = bot
	return t
}

func (t *TimelineUpdater) Parse_RSS() {
	resp, err := http.Get(t.feed)
	if err != nil {
		log.Printf("Error downloading RSS feed:\n")
		log.Printf("%v\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading RSS body:\n")
		log.Printf("%v\n", err)
		return
	}
	rss := new(RSS)
	err = xml.Unmarshal(body, rss)
	if err != nil {
		log.Printf("Error unmarshaling XML from RSS:\n")
		log.Printf("%v\n", err)
		return
	} else if t.last_item.Title == "" {
		t.last_item = rss.Channel.Items[0]
	} else {
		for i := range rss.Channel.Items {
			item := rss.Channel.Items[i]
			if item != t.last_item {
				msg := item.Title + " by " + item.Author
				t.bot.MsgChannel(msg)
				t.bot.MsgChannel(item.Link)
			} else {
				break
			}
		}
		t.last_item = rss.Channel.Items[0]
	}
}

func (t *TimelineUpdater) Run() {
	for {
		t.Parse_RSS()
		time.Sleep(t.interval * time.Second)
	}
}
