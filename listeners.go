package main

import (
	"fmt"
	"github.com/DisgoOrg/disgo/core/events"
	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgolink/lavalink"
	"math/rand"
	"time"
)

var stdCategories = []SegmentCategory{SegmentCategoryIntro, SegmentCategoryOutro, SegmentCategorySponsor}

func checkMusicPlayer(event *events.SlashCommandEvent) *MusicPlayer {
	musicPlayer, ok := musicPlayers[*event.GuildID]
	if !ok {
		_ = event.Create(discord.NewMessageCreateBuilder().SetEphemeral(true).SetContent("No MusicPlayer found for this guild").Build())
		return nil
	}
	return musicPlayer
}

func onSlashCommand(event *events.SlashCommandEvent) {
	switch event.Data.CommandName {
	case "shuffle":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayers == nil {
			return
		}

		if len(musicPlayer.queue) == 0 {
			_ = event.Create(discord.NewMessageCreateBuilder().SetContent("Queue is empty").Build())
			return
		}
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(musicPlayer.queue), func(i, j int) {
			musicPlayer.queue[i], musicPlayer.queue[j] = musicPlayer.queue[j], musicPlayer.queue[i]
		})
		_ = event.Create(discord.NewMessageCreateBuilder().SetContent("Queue shuffled").Build())

	case "filter":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayers == nil {
			return
		}

		flts := musicPlayer.Filters()
		if flts.Timescale() == nil {
			flts.Timescale().Speed = 2
		} else {
			flts.SetTimescale(nil)
		}
		_ = flts.Commit()

	case "queue":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayers == nil {
			return
		}

		if len(musicPlayer.queue) == 0 {
			_ = event.Create(discord.NewMessageCreateBuilder().SetContent("No songs in queue").Build())
		}
		tracks := ""
		for i, track := range musicPlayer.queue {
			tracks += fmt.Sprintf("%d. [%s](%s)\n", i+1, track.Info().Title(), *track.Info().URI())
		}
		_ = event.Create(discord.NewMessageCreateBuilder().SetEmbeds(discord.NewEmbedBuilder().
			SetTitle("Queue:").
			SetDescription(tracks).
			Build(),
		).Build())

	case "pause":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayers == nil {
			return
		}

		pause := !musicPlayer.Paused()
		_ = musicPlayer.Pause(pause)
		message := "paused"
		if !pause {
			message = "resumed"
		}
		_ = event.Create(discord.NewMessageCreateBuilder().SetContent(message + " music").Build())

	case "seek":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayers == nil {
			return
		}

		seconds := *event.Data.Options.Int("seconds")

		if musicPlayer.Track() == nil {
			_ = event.Create(discord.NewMessageCreateBuilder().SetContent("no track playing").Build())
			return
		}
		if seconds > musicPlayer.Track().Info().Length() {
			_ = event.Create(discord.NewMessageCreateBuilder().SetContent("can't seek past the track length").Build())
			return
		}
		_ = musicPlayer.Seek(seconds)
		_ = event.Create(discord.NewMessageCreateBuilder().SetContentf("seeking to %d seconds", seconds).Build())

	case "play":
		voiceState := event.Member.VoiceState()

		if voiceState == nil || voiceState.ChannelID == nil {
			_ = event.Create(discord.NewMessageCreateBuilder().SetEphemeral(true).SetContent("Please join a VoiceChannel to use this command").Build())
			return
		}
		go func() {
			_ = event.DeferCreate(false)

			query := *event.Data.Options.String("query")
			if searchProvider := event.Data.Options.String("search-provider"); searchProvider != nil {
				switch *searchProvider {
				case "yt":
					query = lavalink.SearchTypeYoutube.Apply(query)
				case "ytm":
					query = lavalink.SearchTypeYoutubeMusic.Apply(query)
				case "sc":
					query = lavalink.SearchTypeSoundCloud.Apply(query)
				}
			} else {
				if !URLPattern.MatchString(query) {
					query = lavalink.SearchTypeYoutube.Apply(query)
				}
			}
			musicPlayer, ok := musicPlayers[*event.GuildID]
			if !ok {
				musicPlayer = NewMusicPlayer(*event.GuildID)
				musicPlayers[*event.GuildID] = musicPlayer
			}
			var skipSegments []SegmentCategory
			if option := event.Data.Options.Bool("skip-segments"); option != nil && *option {
				skipSegments = stdCategories
			}

			musicPlayer.Node().RestClient().LoadItemHandler(query, lavalink.NewResultHandler(
				func(track lavalink.Track) {
					if ok = connect(event, voiceState); !ok {
						return
					}
					musicPlayer.Queue(event, skipSegments, track)
				},
				func(playlist lavalink.Playlist) {
					if ok = connect(event, voiceState); !ok {
						return
					}
					musicPlayer.Queue(event, skipSegments, playlist.Tracks...)
				},
				func(tracks []lavalink.Track) {
					if ok = connect(event, voiceState); !ok {
						return
					}
					musicPlayer.Queue(event, skipSegments, tracks[0])
				},
				func() {
					_, _ = event.UpdateOriginal(discord.NewMessageUpdateBuilder().SetContent("no tracks found").Build())
				},
				func(e lavalink.Exception) {
					_, _ = event.UpdateOriginal(discord.NewMessageUpdateBuilder().SetContent("error while loading track:\n" + e.Error()).Build())
				},
			))
		}()
	}
}
