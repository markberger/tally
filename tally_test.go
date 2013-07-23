package tally

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"net/http"
	"testing"
	"encoding/xml"
	"io/ioutil"
)

type server struct {
	recv    []string
	to_send string
	addr    string
}

func mockServer(addr string, to_send string) *server {
	s := new(server)
	s.to_send = to_send
	s.addr = addr
	return s
}

func (bot *Bot) mockConnect(b chan string) {
	dest := bot.Server + ":" + bot.Port
	// Block until the server is ready to accept connections.
	<-b
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		log.Fatalf("Unable to connect to %s\n%v\n\n", dest, err)
	}
	log.Printf("Successfully connected to %s\n", dest)
	bot.conn = conn
	// Block until the server is ready to accept data.
	<-b
	bot.Send("USER " + bot.Nick + " 8 * :" + bot.Nick + "\n")
	bot.Send("NICK " + bot.Nick + "\n")
	bot.Send("JOIN " + bot.Channel + "\n")
}

func (s *server) handle(conn net.Conn, b chan string) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	b <- ""
	for {
		line, err := tp.ReadLine()
		if err != nil {
			fmt.Printf("Error reading line\n")
		}
		s.recv = append(s.recv, line)
		b <- line
		if line == "JOIN #test-channel" {
			if len(s.to_send) != 0 {
				conn.Write([]byte(s.to_send))
			}
		}
	}
}

func (s *server) run(b chan string) {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		for {
			// Signal to bot that the server is ready to start
			// accepting connections.
			b <- ""
			conn, err := l.Accept()
			if err != nil {
				fmt.Printf("%v\n", err)
			} else {
				go s.handle(conn, b)
				l.Close()
				break
			}
		}
	}
}

func runTest(lines int, to_send string) []string {
	InitLogging()
	b := make(chan string)
	s := mockServer("localhost:5000", to_send)
	go s.run(b)
	bot := NewBot()
	bot.mockConnect(b)
	go bot.Run()
	for i := 0; i < lines; i++ {
		<-b
	}
	return s.recv
}

func assertEqual(t *testing.T, s1 string, s2 string) {
	if s1 != s2 {
		t.Errorf("Expecting \"%v\", received: \"%v\"", s2, s1)
	}
}

func TestLoadConfig(t *testing.T) {
	bot := NewBot()
	assertEqual(t, bot.Server, "localhost")
	assertEqual(t, bot.Port, "5000")
	assertEqual(t, bot.Channel, "#test-channel")
	assertEqual(t, bot.Trac_URL, "https://tahoe-lafs.org/trac/tahoe-lafs/")
}

func TestAddAction(t *testing.T) {
	bot := NewBot()
	if len(bot.actions) != 0 {
		t.Errorf("New bot initializes with %v actions. Expected is 0.", len(bot.actions))
	}
	bot.AddAction(`#(\d+)`, ParseTicket, FetchTickets)
	if len(bot.actions) != 1 {
		t.Errorf("AddAction was not successful. Len(bot.actions) == %v", len(bot.actions))
	}
}

func TestConnect(t *testing.T) {
	resp := runTest(3, "")
	assertEqual(t, resp[0], "USER tally 8 * :tally")
	assertEqual(t, resp[1], "NICK tally")
	assertEqual(t, resp[2], "JOIN #test-channel")
}

func TestPING(t *testing.T) {
	resp := runTest(4, "PING test_server\n")
	assertEqual(t, resp[3], "PONG test_server")
}

func TestTicketResponse(t *testing.T) {
	out1 := "PRIVMSG #test-channel :#1382 (immutable peer selection refactoring and enhancements)"
	out2 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382"
	resp := runTest(5, "ticket #1382\n")
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
}

func TestMultipleTickets(t *testing.T) {
	out1 := "PRIVMSG #test-channel :#1382 (immutable peer selection refactoring and enhancements)"
	out2 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382"
	out3 := "PRIVMSG #test-channel :#1057 (Alter mutable files to use servers of happiness)"
	out4 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1057"
	resp := runTest(7, "working on tickets #1382 and #1057\n")
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
	assertEqual(t, resp[5], out3)
	assertEqual(t, resp[6], out4)
}

func TestBadTicketFormat(t *testing.T) {
	out1 := "PRIVMSG #test-channel :#1382 (immutable peer selection refactoring and enhancements)"
	out2 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382"
	out3 := "PRIVMSG #test-channel :#1057 (Alter mutable files to use servers of happiness)"
	out4 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1057"
	resp := runTest(7, "working on tickets #1382 #abc #1182ab and #1057\n")
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
	assertEqual(t, resp[5], out3)
	assertEqual(t, resp[6], out4)
}

func TestNonexistantTicket(t *testing.T) {
	out1 := "PRIVMSG #test-channel :#1382 (immutable peer selection refactoring and enhancements)"
	out2 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382"
	resp := runTest(5, "working on tickets #1000000000 and #1382\n")
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
}

func TestIgnoreList(t *testing.T) {
	out1 := "PRIVMSG #test-channel :#1382 (immutable peer selection refactoring and enhancements)"
	out2 := "PRIVMSG #test-channel :https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382"
	resp := runTest(5, "another-bot: test #13 check...\n I'm working on ticket #1382\n")
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
}

func TestTimelineUpdator(t *testing.T) {
	// Download and parse test RSS feed
	bot := NewBot()
	response, err := http.Get(bot.Trac_RSS)
	if err != nil {
		t.Errorf("Error downloading RSS feed:\n")
		t.Errorf("%v\n", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Error reading RSS body:\n")
		t.Errorf("%v\n", err)
	}
	rss := new(RSS)
	err = xml.Unmarshal(body, rss)

	// Initialize mock server and bot
	InitLogging()
	b := make(chan string)
	s := mockServer("localhost:5000", "")
	go s.run(b)
	bot.mockConnect(b)

	// Set last item to be the 2nd item in the test RSS feed and run
	u := bot.NewTimelineUpdater(bot.Trac_RSS, bot.Interval)
	u.last_item = rss.Channel.Items[1]
	go u.Run()

	// Check to see if the bot sends a message about the first item
	item := rss.Channel.Items[0]
	out1 := "PRIVMSG #test-channel :" + "\"" + item.Title + "\" by " + item.Author
	out2 := "PRIVMSG #test-channel :" + item.Link

	lines := 5
	for i := 0; i < lines; i++ {
		<-b
	}

	resp := s.recv
	assertEqual(t, resp[3], out1)
	assertEqual(t, resp[4], out2)
}
