tally
=====

Tally is an IRC bot that helps you keep track of your open source project. It is implemented as a package in order to provide customization. If you simply want to run an instance of tally, download a release from [here](https://github.com/markberger/tally/releases) or build the example program.


### Features
* __Ticket Recognition:__ Refer to a ticket by #\<ticket_num\> and tally will return information about that ticket.

```
markberger: Can someone help me out with #1382?
tally: #1382 (immutable peer selection refactoring and enhancements)
tally: https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1382
```

* __Tracker Updates:__ Tally will post an update to the channel whenever there is activity on the tracker.

```
tally: Ticket #1057 (Alter mutable files to use servers of happiness) updated by markberger
tally: https://tahoe-lafs.org/trac/tahoe-lafs/ticket/1057#comment:10
```

* __Ignore:__ Running multiple bots? Tally won't respond to input from a set of specified users.

### Supported Trackers
* Trac

