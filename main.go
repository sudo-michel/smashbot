package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const prefix string = "!smashbot"

type Database struct {
	Players []Player `json:"players"`
	Tables  []Table  `json:"tables"`
}

type Player struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	//rajouter d'autre champs
}

type Table struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
}

func loadDatabase() (Database, error) {
	var db Database
	file, err := ioutil.ReadFile("database.json")
	if err != nil {
		if os.IsNotExist(err) {
			return db, nil
		}
		return db, err
	}
	err = json.Unmarshal(file, &db)
	return db, err
}

func saveDatabase(db Database) error {
	file, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("database.json", file, 0644)
}

func addTable(db *Database, numTables int) error {
	for i := 0; i < numTables; i++ {
		newID := fmt.Sprintf("table_%s", uuid.New().String())
		newTable := Table{
			ID:        newID,
			Available: true,
		}
		db.Tables = append(db.Tables, newTable)
	}
	return saveDatabase(*db)
}

func removeTables(db *Database, numTables int) error {
	if numTables > len(db.Tables) {
		return fmt.Errorf("pas assez de tables à supprimer")
	}
	db.Tables = db.Tables[:len(db.Tables)-numTables]
	return saveDatabase(*db)
}

func addPlayer(db *Database, player Player) error {
	for _, p := range db.Players {
		if p.Username == player.Username {
			return fmt.Errorf("player already exists")
		}
	}
	db.Players = append(db.Players, player)
	return saveDatabase(*db)
}

func listPlayers(db Database) string {
	if len(db.Players) == 0 {
		return "No players"
	}
	var playersList strings.Builder
	for i, player := range db.Players {
		playersList.WriteString(fmt.Sprintf("%d. %s\n", i+1, player.Username))
	}
	return playersList.String()
}

func removePlayer(db *Database, username string) error {
	for i, p := range db.Players {
		if p.Username == username {
			db.Players = append(db.Players[:i], db.Players[i+1:]...)
			return saveDatabase(*db)
		}
	}
	return fmt.Errorf("player not found")
}

func sendEmbed(s *discordgo.Session, channelID, title, description string, color int) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erreur lors du chargement du ficher .env")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("Le token du bot n'est pas défini dans le ficher .env")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return

		}

		args := strings.Split(m.Content, " ")
		if args[0] != prefix {
			return
		}

		if args[1] == "hello" {
			s.ChannelMessageSend(m.ChannelID, "world!")
		}

		db, err := loadDatabase()
		if err != nil {
			sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du chargement de la base de données", 0xFF0000)
			return
		}

		switch {
		//pour ajouter un joueur
		case len(args) >= 3 && args[1] == "add" && args[2] == "player":
			if len(args) < 3 || args[1] != "add" || args[2] != "player" {
				sendEmbed(s, m.ChannelID, "Erreur", "Usage: !smashbot help", 0xFF0000)
				return
			}

			newPlayer := Player{
				ID:       m.Author.ID,
				Username: strings.Join(args[3:], " "),
			}

			err = addPlayer(&db, newPlayer)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de l'ajout du joueur: "+err.Error(), 0xFF0000)
				return
			}

			sendEmbed(s, m.ChannelID, "Succès", fmt.Sprintf("Joueur %s ajouté avec succès!", newPlayer.Username), 0x00FF00)

		//pour lister les joueurs
		case len(args) == 3 && args[1] == "list" && args[2] == "player":
			playerList := listPlayers(db)
			sendEmbed(s, m.ChannelID, "Liste des joueurs", playerList, 0x00FF00)

		//pour supprimer un joueur
		case len(args) >= 3 && args[1] == "remove" && args[2] == "player":
			if len(args) < 4 {
				sendEmbed(s, m.ChannelID, "Erreur", "Usage: !smashbot help", 0xFF0000)
				return
			}
			username := strings.Join(args[3:], " ")
			err = removePlayer(&db, username)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de l'ajout du joueur: "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", fmt.Sprintf("Joueur %s supprimé avec succès!", username), 0x00FF00)

		// add a tables
		case args[1] == "add" && args[2] == "tables":
			if len(args) != 4 {
				sendEmbed(s, m.ChannelID, "Erreur", "Usage: !smashbot help", 0xFF0000)
				return
			}
			numTables, err := strconv.Atoi(args[3])
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Nombre de tables invalide", 0xFF0000)
				return
			}
			err = addTable(&db, numTables)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de l'ajout des tables : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", fmt.Sprintf("%d tables ajoutées avec succès!", numTables), 0x00FF00)

			// remove a tables
		case args[1] == "remove" && args[2] == "tables":
			if len(args) != 4 {
				sendEmbed(s, m.ChannelID, "Erreur", "Usage: !smashbot help", 0xFF0000)
				return
			}
			numTables, err := strconv.Atoi(args[3])
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Nombre de tables invalide", 0xFF0000)
				return
			}
			err = removeTables(&db, numTables)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de la suppréssion des tables : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", fmt.Sprintf("%d tables supprimer avec succès!", numTables), 0x00FF00)

		//pour afficher l'aide
		case len(args) == 2 && args[1] == "help":
			sendEmbed(s, m.ChannelID, "Aide", "Commandes disponibles:\n!smashbot add player <username>\n!smashbot remove player <username> \n!smashbot list player \n!smashbot add tables <number> \n!smashbot remove tables <number>", 0x00FF00)

		default:
			sendEmbed(s, m.ChannelID, "Erreur", "Commande non reconnue. Utilisez !smashbot help", 0xFF0000)
		}

	})

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	log.Print("the bot run")

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
