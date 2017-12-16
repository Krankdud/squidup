# SquidUp: Splatoon Pickup Bot
A Discord bot for finding players for league matches and private battles in Splatoon 2.

## Usage
```sh
go get github.com/krankdud/squidup
go install github.com/krankdud/squidup
cd $GOPATH/bin
squidup -token=<Discord bot token>
```

## Commands
* `!register <friend code>` - Registers your friend code.
* `!pair` - Join the queue for pairing with one other person for League battles.
* `!quad` - Join the queue for teaming with three other people for League battles.
* `!private` - Join the queue for a private battle between eight people.
* `!leave` - If you are in a queue, remove yourself from the queue. If you are in a match, remove yourself from the match.