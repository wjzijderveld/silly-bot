package main

import (
	"flag"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	flag.Parse()
	godotenv.Load()

	var err error
	if logger, err = zap.NewProduction(); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	var token string
	if token = os.Getenv("DISCORD_BOT_TOKEN"); token == "" {
		logger.Error("no bot token configured")
		return
	}

	var channelId string
	if channelId = os.Getenv("DISCORD_CHANNEL_ID"); channelId == "" {
		logger.Error("no channelId configured")
		return
	}

	var session *discordgo.Session
	if session, err = discordgo.New("Bot " + token); err != nil {
		logger.Error("failed to initialize session", zap.Error(err))
		return
	}

	go openSession(session)
	defer session.Close()

	// debug
	isOdd := flag.Arg(0) == "odd"

	var deny int64 = 0
	if !isOdd {
		deny = discordgo.PermissionViewChannel
	}

	for session.State.Ready.SessionID == "" {
		logger.Info("waiting to connect")
		time.Sleep(200 * time.Millisecond)
	}

	for _, guild := range session.State.Ready.Guilds {
		logger.Info("connected to guild", zap.String("guildId", guild.ID))
	}

	roleId := "1151602203883221024" // @everyone
	if err = session.ChannelPermissionSet(channelId, roleId, discordgo.PermissionOverwriteTypeRole, 0, deny); err != nil {
		logger.Error("failed to update channel permission", zap.Error(err))
	}

	logger.Info("staying connected...")
	for {
	}
}

func openSession(session *discordgo.Session) {
	if err := session.Open(); err != nil {
		logger.Fatal("failed to open gateway", zap.Error(err))
	}
}
