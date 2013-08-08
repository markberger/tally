// This file contains extra features that aren't enabled by default.

package tally

import (
	"fmt"
	"regexp"
)

// Extract the sender from the given line

func get_user(line string) string {
	re := regexp.MustCompile(`:(.*)!`)
	result := re.FindStringSubmatch(line)
	return result[1]
}

// If someone types "/me <action> tally", tally will
// respond with "/me <action> <user>"

func ParseAction(re *regexp.Regexp, line string) interface{} {
	// Regex: `:\x01ACTION (\w*) <your bot>\x01`
	action := re.FindStringSubmatch(line)
	if len(action) == 0 {
		return nil
	}

	user := get_user(line)
	m := make(map[string] string)
	m["user"] = user
	m["action"] = action[1]
	return m
}

func RespondToAction(bot *Bot, output interface{}) {
	m := output.(map[string]string)
	user := m["user"]
	action := m["action"]
	msg := fmt.Sprintf("\x01ACTION %s %s\x01", action, user)
	bot.MsgChannel(msg)
}