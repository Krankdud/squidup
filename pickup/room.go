package pickup

import (
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var channelMutex sync.Mutex

// Room holds the IDs of channels and the players currently in the room.
type Room struct {
	Channels      []string
	TextChannel   string
	VoiceChannels []string
	Size          int
	Players       []*Player
	Cleaning      bool
}

// AddPlayer adds a player to the room.
func (room *Room) AddPlayer(player *Player) {
	room.Players = append(room.Players, player)
}

// RemovePlayer removes a player from the room.
func (room *Room) RemovePlayer(player *Player) {
	for i, p := range room.Players {
		if p == player {
			copy(room.Players[i:], room.Players[i+1:])
			room.Players[len(room.Players)-1] = nil
			room.Players = room.Players[:len(room.Players)-1]
		}
	}
}

// PlayerCount returns the number of players in the room.
func (room *Room) PlayerCount() int {
	return len(room.Players)
}

// PlayerInRoom checks if a player is in the room.
func (room *Room) PlayerInRoom(player *Player) bool {
	for _, p := range room.Players {
		if p == player {
			return true
		}
	}
	return false
}

// SetupRoom creates the channels for the room.
// session   : Discord session
// queueType : Type of queue. See const.go for values
func (room *Room) SetupRoom(session *discordgo.Session, queueType int) {
	channelMutex.Lock()
	defer channelMutex.Unlock()

	// Create category
	category := room.createCategory(session)

	// Enough players are available to create a room, create channels
	var textChan *discordgo.Channel
	switch queueType {
	case Pair:
		textChan = room.createChannel(session, "pair", "text", category.ID)
		room.createChannel(session, "Pair", "voice", category.ID)
	case Quad:
		textChan = room.createChannel(session, "quad", "text", category.ID)
		room.createChannel(session, "Quad", "voice", category.ID)
	case Private:
		textChan = room.createChannel(session, "private", "text", category.ID)
		room.createChannel(session, "Team Alpha", "voice", category.ID)
		room.createChannel(session, "Team Beta", "voice", category.ID)
	}

	room.sendIntroMessage(session, textChan.ID)

	guildID := GetGuildID(session)

	// Update player roles and flag them as in a match
	for _, player := range room.Players {
		player.IsSearching = false
		player.IsInMatch = true

		switch queueType {
		case Pair:
			session.GuildMemberRoleRemove(guildID, player.ID, RoleSearchPair)
		case Quad:
			session.GuildMemberRoleRemove(guildID, player.ID, RoleSearchQuad)
		case Private:
			session.GuildMemberRoleRemove(guildID, player.ID, RoleSearchPrivate)
		}
		session.GuildMemberRoleAdd(guildID, player.ID, RoleInProgress)
	}
}

// createChannel creates a channel for the room.
// session     : Discord session
// name        : Name of the channel
// channelType : Type of channel. Valid parameters are "text" or "voice"
// categoryID  : Category to place the channel under.
func (room *Room) createChannel(session *discordgo.Session, name, channelType, categoryID string) *discordgo.Channel {
	guildID := GetGuildID(session)
	channel, err := GuildChannelCreateWithParentID(session, guildID, name, channelType, categoryID)
	if err != nil {
		fmt.Println("Error occurred while creating channel: ", err)
		return nil
	}

	// Remove permission from everyone else
	SetRolePermission(session, channel.ID, "@everyone", 0, 1024)
	// Add permissions for the players, bot, and moderators
	for _, player := range room.Players {
		session.ChannelPermissionSet(channel.ID, player.ID, "member", 1024, 0)
	}
	session.ChannelPermissionSet(channel.ID, session.State.User.ID, "member", 1024, 0)
	SetRolePermission(session, channel.ID, "Moderator", 1024, 0)

	room.Channels = append(room.Channels, channel.ID)
	if channelType == "text" {
		room.TextChannel = channel.ID
	} else {
		room.VoiceChannels = append(room.VoiceChannels, channel.ID)
	}

	return channel
}

// createCategory creates a category for the room.
// session     : Discord session
func (room *Room) createCategory(session *discordgo.Session) *discordgo.Channel {
	guildID := GetGuildID(session)
	channel, err := GuildChannelCreateCategory(session, guildID, "Match")
	if err != nil {
		fmt.Println("Error occurred while creating category: ", err)
		return nil
	}

	// Remove permission from everyone else
	SetRolePermission(session, channel.ID, "@everyone", 0, 1024)
	// Add permissions for the players, bot, and moderators
	for _, player := range room.Players {
		session.ChannelPermissionSet(channel.ID, player.ID, "member", 1024, 0)
	}
	session.ChannelPermissionSet(channel.ID, session.State.User.ID, "member", 1024, 0)
	SetRolePermission(session, channel.ID, "Moderator", 1024, 0)

	room.Channels = append(room.Channels, channel.ID)

	return channel
}

// sendIntroMessage outputs the players and their friend codes to the room.
func (room *Room) sendIntroMessage(session *discordgo.Session, channelID string) {
	msg := "Players:"
	for _, player := range room.Players {
		msg += "\n<@" + player.ID + "> - " + player.FriendCode
	}
	msg += "\nType \"!leave\" to leave the room when you are finished.\nGL HF!"

	session.ChannelMessageSend(channelID, msg)
}

// Cleanup deletes a channel after 10 minutes.
func (room *Room) Cleanup(session *discordgo.Session) {
	session.ChannelMessageSend(room.TextChannel, "A player has left. The room will be closed in 10 minutes.")
	time.Sleep(5 * time.Minute)
	session.ChannelMessageSend(room.TextChannel, "Room will be closed in 5 minutes.")
	time.Sleep(4 * time.Minute)
	session.ChannelMessageSend(room.TextChannel, "Room will be closed in 1 minute.")
	time.Sleep(1 * time.Minute)
	channelMutex.Lock()
	for _, c := range room.Channels {
		session.ChannelDelete(c)
	}
	channelMutex.Unlock()
}
