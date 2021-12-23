package main

import (
	"fmt"
	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgo/json"
	"github.com/DisgoOrg/disgolink/lavalink"
)

var _ lavalink.WebsocketMessageInHandler = (*SponsorBlockPlugin)(nil)

type SponsorBlockPlugin struct{}

func (p *SponsorBlockPlugin) OnWebsocketMessageIn(node lavalink.Node, data []byte) bool {
	var opType struct {
		Op   lavalink.OpType    `json:"op"`
		Type lavalink.EventType `json:"type"`
	}
	if err := json.Unmarshal(data, &opType); err != nil {
		node.Lavalink().Logger().Error("failed to unmarshal json", err)
		return true
	}
	if opType.Op != lavalink.OpTypeEvent {
		return false
	}
	switch opType.Type {
	case "SegmentsLoaded":
		var v SegmentsLoadedEvent
		if err := json.Unmarshal(data, &v); err != nil {
			node.Lavalink().Logger().Error("failed to SegmentsLoadedEvent", err)
			return true
		}
		if _, err := musicPlayers[v.GuildID].channel.CreateMessage(discord.NewMessageCreateBuilder().SetContentf("Loaded %d segments", len(v.Segments)).Build()); err != nil {
			node.Lavalink().Logger().Error("failed to send SegmentsLoadedEvent message", err)
		}
		return true

	case "SegmentSkipped":
		var v SegmentSkippedEvent
		if err := json.Unmarshal(data, &v); err != nil {
			node.Lavalink().Logger().Error("failed to SegmentSkipped", err)
			return true
		}
		if _, err := musicPlayers[v.GuildID].channel.CreateMessage(discord.NewMessageCreateBuilder().SetContentf("Skipped `%s` segment from %s to %s", v.Segment.Category, secondsToMinutes(v.Segment.Start/1000), secondsToMinutes(v.Segment.End/1000)).Build()); err != nil {
			node.Lavalink().Logger().Error("failed to send SegmentSkipped message", err)
		}
		return true
	}
	return false
}

func secondsToMinutes(inSeconds int) string {
	return fmt.Sprintf("%d:%d", inSeconds/60, inSeconds%60)
}

type SegmentsLoadedEvent struct {
	GuildID  discord.Snowflake `json:"guildId"`
	Segments []Segment
}

type SegmentSkippedEvent struct {
	GuildID discord.Snowflake `json:"guildId"`
	Segment Segment           `json:"segment"`
}

type Segment struct {
	Category SegmentCategory `json:"category"`
	Start    int             `json:"start"`
	End      int             `json:"end"`
}

type SegmentCategory string

const (
	SegmentCategorySponsor       SegmentCategory = "sponsor"
	SegmentCategorySelfpromo     SegmentCategory = "selfpromo"
	SegmentCategoryInteraction   SegmentCategory = "interaction"
	SegmentCategoryIntro         SegmentCategory = "intro"
	SegmentCategoryOutro         SegmentCategory = "outro"
	SegmentCategoryPreview       SegmentCategory = "preview"
	SegmentCategoryMusicOfftopic SegmentCategory = "music_offtopic"
	SegmentCategoryFiller        SegmentCategory = "filler "
)
