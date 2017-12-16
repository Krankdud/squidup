package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/krankdud/squidup/pickup"
	_ "github.com/mattn/go-sqlite3"
)

var pairQueue, quadQueue, privateQueue pickup.Queue
var rooms []*pickup.Room
var friendCodeRegex *regexp.Regexp
var memberRegex *regexp.Regexp
var database *sql.DB
var playerStore pickup.PlayerStore
var players map[string]*pickup.Player

func init() {
	pairQueue.RequiredPlayers = 2
	quadQueue.RequiredPlayers = 4
	privateQueue.RequiredPlayers = 8
	friendCodeRegex = regexp.MustCompile(`\d{4}-\d{4}-\d{4}`)
	memberRegex = regexp.MustCompile(`<@\d*>`)
	players = make(map[string]*pickup.Player)
}

func main() {
	createDatabase()

	var token string
	flag.StringVar(&token, "token", "", "Discord bot API token")
	flag.Parse()

	if token == "" {
		fmt.Println("Token must be provided to run the bot")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session ", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(presenceUpdate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection ", err)
		return
	}

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
	database.Close()
}

func createDatabase() {
	db, err := sql.Open("sqlite3", "./pickup.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Players (
		ID int,
		DiscordID varchar(255) NOT NULL,
		FriendCode char(14) NOT NULL,
		PRIMARY KEY (ID)
	);`)
	if err != nil {
		log.Fatal(err)
	}

	database = db
	playerStore = pickup.SQLitePlayerStore{DB: db}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	input := strings.Split(m.Content, " ")
	if len(input) < 1 {
		return
	}

	switch command := input[0]; command {
	case "!register":
		if m.ChannelID == pickup.SearchChannelID {
			if len(input) > 1 {
				if friendCodeRegex.MatchString(input[1]) {
					if playerStore.PlayerExists(m.Author.ID) {
						playerStore.UpdateFriendCode(m.Author.ID, input[1])
						s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s>: Your friend code has been updated.", m.Author.ID))
					} else {
						playerStore.Register(m.Author.ID, input[1])
						s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s>: Registered successfully! Use !pair, !quad, or !private to start searching.", m.Author.ID))
					}
				} else {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s>: %s is not a valid friend code. Register by typing \"!register ####-####-####\"", m.Author.ID, input[1]))
				}
			} else {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s>: You must provide your friend code when registering. Register by typing \"!register ####-####-####\"", m.Author.ID))
			}
		}
	case "!pair":
		if m.ChannelID == pickup.SearchChannelID {
			addToQueue(s, m.Author.ID, m.ChannelID, &pairQueue, pickup.Pair)
		}
	case "!quad":
		if m.ChannelID == pickup.SearchChannelID {
			addToQueue(s, m.Author.ID, m.ChannelID, &quadQueue, pickup.Quad)
		}
	case "!private":
		if m.ChannelID == pickup.SearchChannelID {
			addToQueue(s, m.Author.ID, m.ChannelID, &privateQueue, pickup.Private)
		}
	case "!leave":
		if p, ok := players[m.Author.ID]; ok {
			guildID := pickup.GetGuildID(s)
			if p.IsSearching {
				if m.ChannelID == pickup.SearchChannelID {
					p.IsSearching = false
					pairQueue.Remove(p)
					quadQueue.Remove(p)
					privateQueue.Remove(p)
					s.GuildMemberRoleRemove(guildID, m.Author.ID, pickup.RoleSearchPair)
					s.GuildMemberRoleRemove(guildID, m.Author.ID, pickup.RoleSearchQuad)
					s.GuildMemberRoleRemove(guildID, m.Author.ID, pickup.RoleSearchPrivate)
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s>: You have been removed from the queue.", m.Author.ID))
				}
			} else {
				for _, room := range rooms {
					if room.PlayerInRoom(p) && m.ChannelID == room.TextChannel {
						room.RemovePlayer(p)
						p.IsInMatch = false
						s.GuildMemberRoleRemove(guildID, p.ID, pickup.RoleInProgress)
						for _, c := range room.Channels {
							s.ChannelPermissionSet(c, p.ID, "member", 0, 1024)
						}

						for _, p := range room.Players {
							p.IsInMatch = false
							s.GuildMemberRoleRemove(guildID, p.ID, pickup.RoleInProgress)
						}

						if !room.Cleaning {
							room.Cleaning = true
							go room.Cleanup(s)
						}
						break
					}
				}
			}
		}
	}
}

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	// Remove a player from the queue if they go offline
	if p, ok := players[presence.User.ID]; ok {
		if presence.Status == "offline" && p.IsSearching {
			pairQueue.Remove(p)
			quadQueue.Remove(p)
			privateQueue.Remove(p)
			session.GuildMemberRoleRemove(presence.GuildID, p.ID, pickup.RoleSearchPair)
			session.GuildMemberRoleRemove(presence.GuildID, p.ID, pickup.RoleSearchQuad)
			session.GuildMemberRoleRemove(presence.GuildID, p.ID, pickup.RoleSearchPrivate)
		}
	}
}

func addToQueue(s *discordgo.Session, playerID string, channelID string, q *pickup.Queue, queueType int) {
	if playerStore.PlayerExists(playerID) {
		// Check if the player is already searching or in a match
		if p, ok := players[playerID]; ok {
			if p.IsSearching {
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!leave\" your current queue before searching again", playerID))
				return
			} else if p.IsInMatch {
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!leave\" your current match before searching again", playerID))
				return
			}
		} else {
			player := playerStore.GetPlayer(playerID)
			players[playerID] = player
		}

		player := players[playerID]
		player.IsSearching = true

		guildID := pickup.GetGuildID(s)
		// Add appropriate searching role to the player
		switch queueType {
		case pickup.Pair:
			s.GuildMemberRoleAdd(guildID, playerID, pickup.RoleSearchPair)
		case pickup.Quad:
			s.GuildMemberRoleAdd(guildID, playerID, pickup.RoleSearchQuad)
		case pickup.Private:
			s.GuildMemberRoleAdd(guildID, playerID, pickup.RoleSearchPrivate)
		}

		room := q.Enqueue(player)
		if room == nil {
			s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You have been added to the queue", playerID))
		} else {
			room.SetupRoom(s, queueType)
			rooms = append(rooms, room)
		}
	} else {
		s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!register\" before you can search for matches.", playerID))
	}
}

func addTeamToQueue(s *discordgo.Session, playerID string, channelID string, input []string, q *pickup.Queue, queueType int) {
	var team []*pickup.Player

	if !playerStore.PlayerExists(playerID) {
		s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!register\" before you can search for matches.", playerID))
		return
	}

	if p, ok := players[playerID]; ok {
		if p.IsSearching {
			s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!leave\" your current queue before searching again", playerID))
			return
		} else if p.IsInMatch {
			s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: You must \"!leave\" your current match before searching again", playerID))
			return
		}
	} else {
		player := playerStore.GetPlayer(playerID)
		players[playerID] = player
	}

	team = append(team, players[playerID])

	// Make sure each team member can be added to the team
	for i := 1; i < len(input); i++ {
		if memberRegex.MatchString(input[i]) {
			id := input[i][2 : len(input[i])-1]
			if playerStore.PlayerExists(id) {
				if p, ok := players[id]; ok {
					if p.IsSearching {
						s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: %s must \"!leave\" his or her current queue before searching with a team", playerID, input[i]))
						return
					} else if p.IsInMatch {
						s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: %s must \"!leave\" his or her current match before searching with a team", playerID, input[i]))
						return
					}
				} else {
					player := playerStore.GetPlayer(id)
					players[id] = player
				}

				team = append(team, players[id])
			} else {
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: %s needs to be registered before queuing.", playerID, input[i]))
				return
			}
		} else {
			s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: %s is not a valid player name.", playerID, input[i]))
			return
		}
	}

	guildID := pickup.GetGuildID(s)

	// Set roles for each team member
	for _, player := range team {
		players[player.ID] = player
		player.IsSearching = true

		// Add appropriate searching role to the player
		switch queueType {
		case pickup.Pair:
			s.GuildMemberRoleAdd(guildID, player.ID, pickup.RoleSearchPair)
		case pickup.Quad:
			s.GuildMemberRoleAdd(guildID, player.ID, pickup.RoleSearchQuad)
		case pickup.Private:
			s.GuildMemberRoleAdd(guildID, player.ID, pickup.RoleSearchPrivate)
		}
	}

	room := q.EnqueueTeam(team)
	if room == nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>: Your team has been added to the queue.", playerID))
	} else {
		room.SetupRoom(s, queueType)
		rooms = append(rooms, room)
	}
}
