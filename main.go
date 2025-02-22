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
var debug bool

func createLogger() (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}

func main() {
	flag.Parse()
	godotenv.Load()

	debug = os.Getenv("APP_DEBUG") == "true"

	var err error
	if logger, err = createLogger(); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	var token string
	if token = os.Getenv("DISCORD_BOT_TOKEN"); token == "" {
		logger.Error("no bot token configured")
		return
	}

	var interval time.Duration
	if interval, err = time.ParseDuration(os.Getenv("INTERVAL")); err != nil {
		logger.Error("invalid interval configured, it needs to be parseable by time.ParseDuration, in the future and less than 12 hours", zap.Error(err))
		return
	}

	if interval < 0 || interval > 12*time.Hour {
		logger.Error("invalid interval configured, it needs to be parseable by time.ParseDuration, in the future and less than 12 hours", zap.Error(err))
		return
	}

	var session *discordgo.Session
	if session, err = discordgo.New("Bot " + token); err != nil {
		logger.Error("failed to initialize session", zap.Error(err))
		return
	}

	openSession(session)
	defer session.Close()

	// based on configured interval, find first reasonable point in time to do the flip
	if interval >= time.Minute { // for debugging, allows 5s intervals
		truncate := time.Hour
		if interval < time.Hour {
			truncate = time.Minute
		}
		firstTick := time.Now().Add(truncate).Truncate(truncate)

		untilFirstTick := time.Until(firstTick)
		timer := time.NewTimer(untilFirstTick)
		logger.Info("waiting for first tick", zap.Duration("duration", untilFirstTick.Truncate(time.Second)))
		<-timer.C
	}

	logger.Info("first tick, updating visibility and starting timer")
	flipVisibility(session)

	startTimer(interval, session)
}

func startTimer(d time.Duration, session *discordgo.Session) {
	timer := time.NewTimer(d)

	for {
		<-timer.C
		logger.Debug("next tick")
		flipVisibility(session)
		timer.Reset(d)
	}
}

func flipVisibility(session *discordgo.Session) {
	var channel *discordgo.Channel
	var err error
	for _, mc := range managedChannels {
		if channel, err = session.Channel(mc.ChannelId); err != nil {
			logger.Error("failed to get channel info during flip", zap.Error(err))
			continue
		}

		var allow, deny int64
		if allow, deny, err = determineNextState(mc, channel.PermissionOverwrites); err != nil {
			logger.Error("failed to determine next state", zap.Error(err))
			continue
		}

		if err = session.ChannelPermissionSet(mc.ChannelId, mc.GuildId, discordgo.PermissionOverwriteTypeRole, allow, deny); err != nil {
			logger.Error("failed to update permissions", zap.Error(err))
		}
	}
}

func determineNextState(managedChannel ManagedChannel, currentOverwrites []*discordgo.PermissionOverwrite) (int64, int64, error) {
	// default behaviour is to deny ViewChannel only
	var allow int64 = 0
	var deny int64 = discordgo.PermissionViewChannel

	for _, overwrite := range currentOverwrites {
		if overwrite.ID != managedChannel.GuildId {
			// we're only interested in the @everyone overwrites
			continue
		}

		allow = overwrite.Allow
		deny = overwrite.Deny

		logger.Debug(
			"current permissions",
			zap.Bool("allowed", allow&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel),
			zap.Bool("denied", deny&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel),
		)
		if allow&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel {
			// currently allowed, change to denied
			allow = allow ^ discordgo.PermissionViewChannel
			deny = deny | discordgo.PermissionViewChannel
		} else {
			allow = allow | discordgo.PermissionViewChannel
			deny = deny ^ discordgo.PermissionViewChannel
		}

		// found our overwrite, we don't have to continue the loop
		break
	}

	logger.Debug(
		"next state determined",
		zap.Bool("allowed", allow&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel),
		zap.Bool("denied", deny&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel),
	)

	return allow, deny, nil
}

func openSession(session *discordgo.Session) {
	session.AddHandlerOnce(func(session *discordgo.Session, _ *discordgo.Connect) {
		logger.Info("connected to the Discord gateway \\o/")

		var err error
		for _, guild := range session.State.Guilds {
			logger.Info("connected to guild", zap.String("guildId", guild.ID))

			var channels []*discordgo.Channel
			if channels, err = session.GuildChannels(guild.ID); err != nil {
				logger.Error("failed to get channels for guild", zap.String("guildId", guild.ID), zap.String("name", guild.Name), zap.Error(err))
				continue
			}

			for _, channel := range channels {
				if channel.Type != discordgo.ChannelTypeGuildText {
					// we only care about regular text channels
					continue
				}

				logger.Info("channel info", zap.String("name", channel.Name), zap.Any("permissionOverwrites", channel.PermissionOverwrites))

				var hasManage, hasView = false, false
				for _, overwrite := range channel.PermissionOverwrites {
					if overwrite.ID != session.State.User.ID {
						continue
					}
					if overwrite.Allow&discordgo.PermissionManageChannels == discordgo.PermissionManageChannels {
						hasManage = true
					}
					if overwrite.Allow&discordgo.PermissionViewChannel == discordgo.PermissionViewChannel {
						hasView = true
					}
				}

				if hasManage && hasView {
					logger.Info("we have VIEW_CHANNEL & MANAGE_CHANNEL/PERMISSIONS permissions \\o/", zap.String("channel", channel.Name))
					managedChannels = append(managedChannels, ManagedChannel{
						GuildId:   channel.GuildID,
						ChannelId: channel.ID,
					})
				}

				if hasManage && !hasView {
					logger.Info("we can manage a channel, but we also need VIEW_CHANNEL to be able to flip the permissions", zap.String("channel", channel.Name))
				}
			}

			if len(managedChannels) == 0 {
				logger.Info("no managed channels found, exiting")
				os.Exit(0)
			}
		}
	})

	if err := session.Open(); err != nil {
		logger.Fatal("failed to open gateway", zap.Error(err))
	}

	logger.Info("connecting to Discord gateway...")
}
