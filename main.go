package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const prefix string = "!smashbot"

type Database struct {
	Players     []Player     `json:"players"`
	Tables      []Table      `json:"tables"`
	Tournaments []Tournament `json:"tournaments"`
}
type Round struct {
	Matches []Match `json:"matches"`
}
type Player struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	//rajouter d'autre champs
}

type Table struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
	MatchID   string `json:"match_id"`
}

type Tournament struct {
	ID           string   `json:"id"`
	Matches      []Match  `json:"matches"`
	Rounds       []Round  `json:"rounds"`
	Players      []string `json:"players"`
	Status       TournamentStatus
	CurrentRound int `json:"current_round"`
}

type Match struct {
	ID      string `json:"id"`
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	Winner  string `json:"winner"`
	TableID string `json:"table_id"`
}

type TournamentStatus string

const (
	TournamentStatusPending  TournamentStatus = "pending"
	TournamentStatusOngoing  TournamentStatus = "ongoing"
	TournamentStatusComplete TournamentStatus = "complete"
)

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

func startTournament(db *Database) (*Tournament, error) {
	if len(db.Players) < 2 {
		return nil, fmt.Errorf("pas assez de joueurs pour démarrer un tournoi")
	}

	if len(db.Tables) == 0 {
		return nil, fmt.Errorf("aucune table disponible")
	}

	tournament := &Tournament{
		ID:           fmt.Sprintf("T%d", len(db.Tournaments)+1),
		CurrentRound: 0,
		Status:       TournamentStatusPending,
	}

	players := make([]string, len(db.Players))
	for i, player := range db.Players {
		players[i] = player.Username
	}
	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })
	tournament.Players = players

	// Créer le premier tour
	firstRound := createRound(players, db.Tables)
	tournament.Rounds = append(tournament.Rounds, firstRound)
	tournament.Status = TournamentStatusOngoing

	db.Tournaments = append(db.Tournaments, *tournament)
	return tournament, saveDatabase(*db)
}

func createRound(players []string, tables []Table) Round {
	round := Round{}
	tableIndex := 0

	for i := 0; i < len(players); i += 2 {
		match := Match{
			ID:      fmt.Sprintf("M%d", len(round.Matches)+1),
			Player1: players[i],
			TableID: tables[tableIndex].ID,
		}
		if i+1 < len(players) {
			match.Player2 = players[i+1]
		} else {
			// Si c'est un joueur seul, il passe automatiquement au tour suivant
			match.Winner = players[i]
		}
		round.Matches = append(round.Matches, match)
		tableIndex = (tableIndex + 1) % len(tables)
	}

	return round
}

func updateTournament(db *Database, tournamentID string) error {
	for i, t := range db.Tournaments {
		if t.ID == tournamentID {
			currentRound := t.Rounds[t.CurrentRound]
			allMatchesComplete := true
			winners := []string{}

			for _, match := range currentRound.Matches {
				if match.Winner == "" {
					allMatchesComplete = false
					break
				}
				winners = append(winners, match.Winner)
			}

			if allMatchesComplete {
				if len(winners) == 1 {
					// Le tournoi est terminé
					db.Tournaments[i].Status = TournamentStatusComplete
				} else {
					// Créer le prochain tour
					nextRound := createRound(winners, db.Tables)
					db.Tournaments[i].Rounds = append(db.Tournaments[i].Rounds, nextRound)
					db.Tournaments[i].CurrentRound++
				}
			}

			return saveDatabase(*db)
		}
	}

	return fmt.Errorf("tournoi non trouvé")
}

func updateMatchResult(db *Database, matchID string, winnerName string) error {
	for i, tournament := range db.Tournaments {
		for j, round := range tournament.Rounds {
			for k, match := range round.Matches {
				if match.ID == matchID {
					// Normaliser les noms pour la comparaison
					normalizedWinner := strings.ToLower(strings.TrimSpace(winnerName))
					normalizedPlayer1 := strings.ToLower(strings.TrimSpace(match.Player1))
					normalizedPlayer2 := strings.ToLower(strings.TrimSpace(match.Player2))

					if normalizedWinner != normalizedPlayer1 && normalizedWinner != normalizedPlayer2 {
						return fmt.Errorf("le gagnant doit être l'un des joueurs du match: %s ou %s", match.Player1, match.Player2)
					}

					// Utiliser le nom original du joueur pour la mise à jour
					if normalizedWinner == normalizedPlayer1 {
						db.Tournaments[i].Rounds[j].Matches[k].Winner = match.Player1
					} else {
						db.Tournaments[i].Rounds[j].Matches[k].Winner = match.Player2
					}

					return updateTournament(db, tournament.ID)
				}
			}
		}
	}
	return fmt.Errorf("match non trouvé")
}

func getTournamentStatus(db Database) string {
	if len(db.Tournaments) == 0 {
		return "Aucun tournoi en cours."
	}

	tournament := db.Tournaments[len(db.Tournaments)-1]
	status := fmt.Sprintf("État du tournoi (ID: %s):\n", tournament.ID)
	status += fmt.Sprintf("Statut: %s\n", tournament.Status)
	status += fmt.Sprintf("Tour actuel: %d\n\n", tournament.CurrentRound+1)

	currentRound := tournament.Rounds[tournament.CurrentRound]
	status += "Matchs en cours:\n"
	for _, match := range currentRound.Matches {
		status += fmt.Sprintf("- Match %s: %s vs %s", match.ID, match.Player1, match.Player2)
		if match.Winner != "" {
			status += fmt.Sprintf(" (Gagnant: %s)", match.Winner)
		}
		status += "\n"
	}

	return status
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
				ID:       uuid.New().String(),
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

		case len(args) == 3 && args[1] == "tournament" && args[2] == "start":
			tournament, err := startTournament(&db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du démarrage du tournoi : "+err.Error(), 0xFF0000)
				return
			}

			var matchesInfo strings.Builder
			matchesInfo.WriteString(fmt.Sprintf("Tournoi ID: %s\n\n", tournament.ID))
			matchesInfo.WriteString("Liste des joueurs:\n")
			for i, player := range tournament.Players {
				matchesInfo.WriteString(fmt.Sprintf("%d. %s\n", i+1, player))
			}
			matchesInfo.WriteString("\nMatches du premier tour:\n")
			for _, match := range tournament.Rounds[0].Matches {
				if match.Player2 != "" {
					matchesInfo.WriteString(fmt.Sprintf("Match ID: %s - %s vs %s (Table: %s)\n",
						match.ID, match.Player1, match.Player2, match.TableID))
				} else {
					matchesInfo.WriteString(fmt.Sprintf("Match ID: %s - %s passe automatiquement (Table: %s)\n",
						match.ID, match.Player1, match.TableID))
				}
			}

			sendEmbed(s, m.ChannelID, "Tournoi démarré", matchesInfo.String(), 0x00FF00)

		case len(args) == 5 && args[1] == "match" && args[2] == "result":
			matchID := args[3]
			winnerName := args[4]
			err := updateMatchResult(&db, matchID, winnerName)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de la mise à jour du résultat : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "Résultat du match mis à jour avec succès!", 0x00FF00)

		//pour afficher l'état du tournoi
		case len(args) == 3 && args[1] == "tournament" && args[2] == "status":
			status := getTournamentStatus(db)
			sendEmbed(s, m.ChannelID, "État du tournoi", status, 0x00FF00)

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
			helpMessage := `Commandes disponibles:
Gestion des joueurs:
  !smashbot add player <username>    : Ajoute un nouveau joueur
  !smashbot remove player <username> : Supprime un joueur existant
  !smashbot list player              : Affiche la liste des joueurs

Gestion des tables:
  !smashbot add tables <number>    : Ajoute un certain nombre de tables
  !smashbot remove tables <number> : Supprime un certain nombre de tables

Gestion des tournois:
  !smashbot tournament start : Démarre un nouveau tournoi
  !smashbot match result <match_id> <winner_name> : Enregistre le résultat d'un match


Autres:
  !smashbot help  : Affiche ce message d'aide

Note: Assurez-vous d'utiliser les IDs appropriés pour les tournois, tables et joueurs lors de l'utilisation des commandes.`

			sendEmbed(s, m.ChannelID, "Aide", helpMessage, 0x00FF00)
		default:
			sendEmbed(s, m.ChannelID, "Erreur", "Commande non reconnue. Utilisez !smashbot help", 0xFF0000)
		}

	})

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer func(sess *discordgo.Session) {
		err := sess.Close()
		if err != nil {

		}
	}(sess)

	log.Print("the bot run")

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
