package main

import (
  "github.com/markberger/tally"
)

func main() {
	tally.InitLogging()
	bot := tally.NewBot()
	bot.Connect()
	bot.Run()
}