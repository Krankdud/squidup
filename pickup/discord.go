package pickup

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// GetGuildID gets the guild ID from the session.
func GetGuildID(s *discordgo.Session) string {
	return s.State.Guilds[0].ID
}

// SetRolePermission sets the permission of a role in a channel.
func SetRolePermission(s *discordgo.Session, channelID string, roleName string, allow int, deny int) {
	guildID := GetGuildID(s)
	roles, _ := s.GuildRoles(guildID)
	for _, role := range roles {
		if role.Name == roleName {
			s.ChannelPermissionSet(channelID, role.ID, "role", allow, deny)
		}
	}
}

// GuildChannelCreateWithParentID creates a new channel in a given guild under a given parentID
// guildID   : The ID of a Guild
// name      : Name of the channel (2-100 chars length)
// ctype     : Type of the channel (voice or text)
// parentID  : The ID of the parent
func GuildChannelCreateWithParentID(s *discordgo.Session, guildID, name, ctype, parentID string) (st *discordgo.Channel, err error) {

	data := struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		ParentID string `json:"parent_id"`
	}{name, ctype, parentID}

	body, err := s.RequestWithBucketID("POST", discordgo.EndpointGuildChannels(guildID), data, discordgo.EndpointGuildChannels(guildID))
	if err != nil {
		return
	}

	err = unmarshal(body, &st)
	return
}

// GuildChannelCreateCategory creates a new category in a given guild under a given parentID
// guildID   : The ID of a Guild
// name      : Name of the category (2-100 chars length)
func GuildChannelCreateCategory(s *discordgo.Session, guildID, name string) (st *discordgo.Channel, err error) {

	data := struct {
		Name string `json:"name"`
		Type int    `json:"type"`
	}{name, 4}

	body, err := s.RequestWithBucketID("POST", discordgo.EndpointGuildChannels(guildID), data, discordgo.EndpointGuildChannels(guildID))
	if err != nil {
		return
	}

	err = unmarshal(body, &st)
	return
}

func unmarshal(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return discordgo.ErrJSONUnmarshal
	}

	return nil
}
