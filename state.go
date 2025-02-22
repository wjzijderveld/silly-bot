package main

type ManagedChannel struct {
	GuildId   string
	ChannelId string
}

var managedChannels []ManagedChannel
