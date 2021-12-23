package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/DisgoOrg/disgo/core"
	"github.com/DisgoOrg/disgo/core/bot"
	"github.com/DisgoOrg/disgo/core/events"
	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgo/gateway"
	"github.com/DisgoOrg/disgolink/lavalink"
	"github.com/DisgoOrg/log"
)

var (
	URLPattern = regexp.MustCompile("^https?://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]?")

	token        = os.Getenv("disgolink_token")
	guildID      = discord.Snowflake(os.Getenv("guild_id"))
	disgo        *core.Bot
	dgolink      lavalink.Disgolink
	musicPlayers = map[discord.Snowflake]*MusicPlayer{}
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetLevel(log.LevelDebug)
	log.Info("starting _example...")

	var err error
	disgo, err = bot.New(token,
		bot.WithGatewayOpts(gateway.WithGatewayIntents(discord.GatewayIntentGuilds|discord.GatewayIntentGuildVoiceStates)),
		bot.WithCacheOpts(core.WithCacheFlags(core.CacheFlagsDefault)),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnSlashCommand: onSlashCommand,
		}),
	)
	if err != nil {
		log.Fatalf("error while building disgo instance: %s", err)
		return
	}

	defer func() {
		if err = disgo.Close(context.TODO()); err != nil {
			log.Error("error while closing disgo instance: %s", err)
		}
	}()

	dgolink = lavalink.NewDisgolink(disgo, lavalink.WithPlugins(&SponsorBlockPlugin{}))
	registerNodes()

	defer dgolink.Close(context.TODO())

	_, err = disgo.SetGuildCommands(guildID, commands)
	if err != nil {
		log.Errorf("error while registering guild commands: %s", err)
	}

	err = disgo.ConnectGateway(context.TODO())
	if err != nil {
		log.Fatalf("error while connecting to discord: %s", err)
	}

	log.Infof("_example is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func connect(event *events.SlashCommandEvent, voiceState *core.VoiceState) bool {
	err := voiceState.Channel().Connect(context.TODO())
	if err != nil {
		_, _ = event.UpdateOriginal(discord.NewMessageUpdateBuilder().SetContent("error while connecting to channel:\n" + err.Error()).Build())
		log.Errorf("error while connecting to channel: %s", err)
		return false
	}
	return true
}

func registerNodes() {
	secure, _ := strconv.ParseBool(os.Getenv("lavalink_secure"))
	dgolink.AddNode(lavalink.NodeConfig{
		Name:     "test",
		Host:     os.Getenv("lavalink_host"),
		Port:     os.Getenv("lavalink_port"),
		Password: os.Getenv("lavalink_password"),
		Secure:   secure,
	})
}
