package tally

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// The Action struct represents a possible reaction to a given line.
// Regexps are attached to the Action so that they can be compiled once
// when the bot is initialized instead of each time a line needs to be parsed.

type Action struct {
	re    *regexp.Regexp
	Parse func(*regexp.Regexp, string) interface{}
	Run   func(*Bot, interface{})
}

type Bot struct {
	Server   string
	Port     string
	Nick     string
	Channel  string
	Trac_URL string
	Tickets  map[string]bool
	Trac_RSS string
	Interval time.Duration
	Ignore   []string
	conn     net.Conn

	actions []*Action
}

// Returns a new bot that has been configured according to
// config.json

func NewBot() *Bot {
	bot := new(Bot)
	f, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal("Error loading config.json: %v\n\n", err)
	}
	err = json.Unmarshal(f, bot)
	if err != nil {
		log.Fatal("Error unmarshalling config.json: %v\n\n", err)
	}
	bot.Tickets = make(map[string]bool)
	return bot
}

// Establishes a connection to the server and joins a channel

func (bot *Bot) Connect() {
	dest := bot.Server + ":" + bot.Port
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		log.Fatalf("Unable to connect to %s\nError: %v\n\n", dest, err)
	}
	log.Printf("Successfully connected to %s\n", dest)
	bot.conn = conn
	bot.Send("USER " + bot.Nick + " 8 * :" + bot.Nick + "\n")
	bot.Send("NICK " + bot.Nick + "\n")
	bot.Send("JOIN " + bot.Channel + "\n")
}

// Sends a string to the server. Strings should end with a
// newline character.

func (bot *Bot) Send(str string) {
	msg := []byte(str)
	_, err := bot.conn.Write(msg)
	if err != nil {
		log.Printf("Error sending: %s", msg)
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Successfully sent: %s", msg)
	}
}

// Sends a message to the channel

func (bot *Bot) MsgChannel(line string) {
	bot.Send("PRIVMSG " + bot.Channel + " :" + line + "\n")
}

func (bot *Bot) PrivateMsg(user string, line string) {
	bot.Send("PRIVMSG " + user + " :" + line + "\n")
}

func (bot *Bot) parse(line string) {
	for i := range bot.Ignore {
		if strings.Contains(line, bot.Ignore[i]) {
			return
		}
	}
	for i := range bot.actions {
		action := bot.actions[i]
		resp := action.Parse(action.re, line)
		if resp != nil {
			go action.Run(bot, resp)
		}
	}
}

func (bot *Bot) AddAction(re string, parse func(*regexp.Regexp, string) interface{},
	run func(*Bot, interface{})) {
	action := new(Action)
	action.re = regexp.MustCompile(re)
	action.Parse = parse
	action.Run = run
	bot.actions = append(bot.actions, action)
}

func signalHandling(bot *Bot) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		if sig == os.Interrupt {
			log.Printf("Bot received os.Interrupt, exiting normally.\n\n")
			bot.Send("QUIT :\n")
			bot.conn.Close()
			os.Exit(0)
		}
	}
}

func (bot *Bot) Run() {
	go signalHandling(bot)
	bot.SetActions()
	t := bot.NewTimelineUpdater(bot.Trac_RSS, bot.Interval)
	go t.Run()
	reader := bufio.NewReader(bot.conn)
	tp := textproto.NewReader(reader)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			log.Printf("Error reading line: %s\n", line)
			log.Fatalf("Error: %v\n", err)
		} else {
			bot.parse(line)
		}
	}
}

func ParseTicket(re *regexp.Regexp, line string) interface{} {
	//Regex: #(\d+)([:alpha:])*
	matches := re.FindAllStringSubmatch(line, 10)
	var ticket_nums []string
	for i := range matches {
		if matches[i][2] == "" {
			ticket_nums = append(ticket_nums, matches[i][1])
		}
	}
	if len(ticket_nums) == 0 {
		return nil
	}
	return ticket_nums
}

func removeTicket(bot *Bot, num string) {
	time.Sleep(5 * time.Minute)
	bot.Tickets[num] = false
}

func FetchTickets(bot *Bot, tickets interface{}) {
	url := bot.Trac_URL + "ticket/"
	ticket_nums := reflect.ValueOf(tickets)
	for i := 0; i < ticket_nums.Len(); i++ {
		num := ticket_nums.Index(i).String()

		// Check to see if we gave a link to this ticket recently.
		if bot.Tickets[num] == false {
			bot.Tickets[num] = true
			go removeTicket(bot, num)
		} else {
			continue
		}

		dest := url + num
		resp, err := http.Get(dest)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		s := string(body)
		if strings.Contains(s, "<h1>Error:") {

		} else {
			s = s[:300]
			a := strings.Split(s, "\n")
			title := strings.TrimLeft(a[8], " ")
			bot.MsgChannel(title)
			bot.MsgChannel(dest)
		}
	}
}

func ParsePing(re *regexp.Regexp, line string) interface{} {
	//Regex: ^PING
	str := re.FindString(line)
	if str == "" {
		return nil
	}
	return line
}

func SendPong(bot *Bot, str interface{}) {
	line := reflect.ValueOf(str).String()
	log.Printf("Received: %s\n", line)
	strs := strings.Split(line, " ")
	bot.Send("PONG " + strs[1] + "\n")
}

func (bot *Bot) SetActions() {
	if bot.Trac_URL != "" {
		bot.AddAction(`#(\d+)([:alpha:])*`, ParseTicket, FetchTickets)
	}
	bot.AddAction(`^PING`, ParsePing, SendPong)
}

func InitLogging() {
	logf, err := os.OpenFile("bot_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0667)
	if err != nil {
		log.Fatal("Cannot open/create log file.\nError: %v\n\n", err)
	}
	log.SetOutput(logf)
}
