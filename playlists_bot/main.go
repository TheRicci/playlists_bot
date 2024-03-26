package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Bot struct {
	*discordgo.Session
	*database
	randomMap         map[string]*[]videoQuery //no need to use mutex, every key will only be accessed by one user from one command
	openCommandRandom map[string]*discordgo.Message
	openCommandSearch map[string]MenuSelectionState
}

type MenuSelectionState struct {
	currentIndex int
	maxIndex     int
	videos       []videoQuery
	list         []*[]discordgo.SelectMenuOption
}

func newBot() Bot {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal().Msgf("error while instantiating Discord bot : %s ", err.Error())
	}

	return Bot{
		dg,
		newDB(),
		make(map[string]*[]videoQuery),
		make(map[string]*discordgo.Message),
		make(map[string]MenuSelectionState),
	}
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal().Msgf("Error loading .env file")
	}

	bot := newBot()
	err = bot.Open()
	if err != nil {
		log.Fatal().Msgf("error while opening connection with Discord : %s ", err.Error())
	}

	log.Info().Msg("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := bot.ApplicationCommandCreate(bot.State.User.ID, "", v)
		if err != nil {
			log.Panic().Msgf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	bot.AddHandler(bot.interactionHandler)
	bot.AddHandler(bot.MessageHandler)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	log.Info().Msg("Removing commands...")
	for _, v := range registeredCommands {
		err := bot.ApplicationCommandDelete(bot.State.User.ID, "", v.ID)
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	_ = bot.Close()
	log.Info().Msg("Bot exiting")
}
