tally
=====

Tally is an IRC bot that helps you keep track of your open source project. It is implemented as a package in order to provide customization. If you simply want to run an instance of tally, download a release from [here](https://github.com/markberger/tally/releases) or build the following Go program:

```go
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
```

### Supported Trackers
* Trac

### Features
* Ticket Recognition: Refer to a ticket by #\<ticket_num\> and tally will return information about that ticket.
* Ignore: Running multiple bots? Tally won't respond to input from a set of specified users.

