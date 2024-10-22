package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const prefix string = "!smashbot"

// Basic structure that stores all data
type Database struct {
	Players     []Player     `json:"players"`
	Tables      []Table      `json:"tables"`
	Tournaments []Tournament `json:"tournament"`
}

type Round struct {
	Matches []Match `json:"matches"`
}
type Player struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Table struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
	MatchID   string `json:"match_id"`
}

type Tournament struct {
	ID           string                     `json:"id"`
	Matches      []Match                    `json:"matches"`
	Rounds       []Round                    `json:"rounds"`
	Players      []string                   `json:"player_ids"`
	Status       TournamentStatus           `json:"status"`
	CurrentRound int                        `json:"current_round"`
	Stages       map[string]map[string]bool `json:"stages"`
}

type Match struct {
	ID      string   `json:"id"`
	Players []string `json:"players"`
	Player1 string   `json:"player1"`
	Player2 string   `json:"player2"`
	Winner  string   `json:"winner"`
	TableID string   `json:"table_id"`
}

type TournamentStatus string

const (
	TournamentStatusPending  TournamentStatus = "pending"
	TournamentStatusOngoing  TournamentStatus = "ongoing"
	TournamentStatusComplete TournamentStatus = "complete"
)

// Loads or creates new database from file
func loadDatabase() (*Database, error) {
	file, err := os.ReadFile("database.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Database file does not exist. Creating a new one.")
			return &Database{
				Players:     []Player{},
				Tables:      []Table{},
				Tournaments: []Tournament{},
			}, nil
		}
		log.Printf("Error reading database file: %v", err)
		return nil, fmt.Errorf("error reading database file: %w", err)
	}

	var db Database
	if err := json.Unmarshal(file, &db); err != nil {
		log.Printf("Error unmarshalling database: %v", err)
		return nil, fmt.Errorf("error unmarshalling database: %w", err)
	}

	if db.Players == nil {
		db.Players = []Player{}
	}
	if db.Tables == nil {
		db.Tables = []Table{}
	}
	if db.Tournaments == nil {
		db.Tournaments = []Tournament{}
	}

	log.Println("Database loaded successfully")
	return &db, nil
}

// Saves current database state to file
func saveDatabase(db Database) error {
	file, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		log.Printf("Error marshalling database: %v", err)
		return fmt.Errorf("error marshalling database: %w", err)
	}

	err = os.WriteFile("database.json", file, 0644)
	if err != nil {
		log.Printf("Error writing database file: %v", err)
		return fmt.Errorf("error writing database file: %w", err)
	}

	log.Println("Database saved successfully")
	return nil
}

/* Player management functions */

// Adds new player to database
func addPlayer(db *Database, player Player) error {
	// Vérifier si le joueur existe déjà
	for _, p := range db.Players {
		if p.Username == player.Username {
			return fmt.Errorf("player already exists")
		}
	}
	db.Players = append(db.Players, player)
	return saveDatabase(*db)
}

// Removes player from database
func removePlayer(db *Database, username string) error {
	for i, p := range db.Players {
		if p.Username == username {
			// Supprimer le joueur en utilisant append
			db.Players = append(db.Players[:i], db.Players[i+1:]...)
			return saveDatabase(*db)
		}
	}
	return fmt.Errorf("player not found")
}

// Lists all players in the database
func listPlayers(db *Database) string {
	if len(db.Players) == 0 {
		return "No players"
	}
	var playersList strings.Builder
	for i, player := range db.Players {
		playersList.WriteString(fmt.Sprintf("%d. %s\n", i+1, player.Username))
	}
	return playersList.String()
}

// Adds new table to database
func addTable(db *Database, numTables int) error {
	for i := 0; i < numTables; i++ {
		newTable := Table{
			ID:        uuid.New().String(),
			Available: true,
		}
		db.Tables = append(db.Tables, newTable)
	}
	return saveDatabase(*db)
}

// Removes table from database
func removeTables(db *Database, numTables int) error {
	if numTables > len(db.Tables) {
		return fmt.Errorf("pas assez de tables à supprimer")
	}
	db.Tables = db.Tables[:len(db.Tables)-numTables]
	return saveDatabase(*db)
}

// Returns the current tournament
func getCurrentTournament(db *Database) *Tournament {
	if len(db.Tournaments) == 0 {
		return nil
	}
	return &db.Tournaments[len(db.Tournaments)-1]
}

// Updates the database with the current tournament
func getTournamentStatus(db Database) string {
	tournament := getCurrentTournament(&db)
	if tournament == nil {
		return "Aucun tournoi en cours."
	}

	if tournament.Status == TournamentStatusComplete {
		return "Aucun tournoi en cours. Le dernier tournoi est terminé."
	}

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

// Starts a new tournament
func startTournament(db *Database) error {
	if len(db.Players) < 2 {
		return fmt.Errorf("pas assez de joueurs pour démarrer un tournoi")
	}

	if len(db.Tables) == 0 {
		return fmt.Errorf("aucune table disponible")
	}

	tournament := Tournament{
		ID:           fmt.Sprintf("T%d", len(db.Tournaments)+1),
		CurrentRound: 0,
		Status:       TournamentStatusPending,
		Players:      make([]string, 0),
		Stages:       make(map[string]map[string]bool),
	}

	for _, p := range db.Players {
		tournament.Players = append(tournament.Players, p.Username)
	}

	rand.Shuffle(len(tournament.Players), func(i, j int) {
		tournament.Players[i], tournament.Players[j] = tournament.Players[j], tournament.Players[i]
	})

	firstRound := Round{
		Matches: createMatches(tournament.Players),
	}

	tournament.Rounds = append(tournament.Rounds, firstRound)
	tournament.Status = TournamentStatusOngoing

	db.Tournaments = append(db.Tournaments, tournament)
	return saveDatabase(*db)
}

// Creates matches for the first round of the tournament
func createMatches(player []string) []Match {
	var (
		matches = []Match{}
	)
	evenCount := nearestEvenNumber(len(player))

	for i := 0; i < evenCount; i += 2 {
		matches = append(matches, Match{
			ID:      "",
			Players: nil,
			Player1: player[i],
			Player2: player[i+1],
			Winner:  "",
			TableID: "",
		})
	}
	for i := evenCount; i < len(player); i++ {
		matches = append(matches, Match{
			Player1: player[i],
		})
	}

	return matches
}

// Creates a new round for the tournament
func createRound(tournament *Tournament, tables []Table) Round {
	round := Round{}
	if len(tables) == 0 {
		return round
	}

	tableIndex := 0
	currentStage := fmt.Sprintf("stage%d", tournament.CurrentRound+1)
	nextStage := fmt.Sprintf("stage%d", tournament.CurrentRound+2)

	playersMap := tournament.Stages[currentStage]
	players := make([]string, 0, len(playersMap))
	for player := range playersMap {
		players = append(players, player)
	}

	for len(players) > 0 && tableIndex < len(tables) {
		match := Match{
			ID:      fmt.Sprintf("M%d", len(round.Matches)+1),
			Player1: players[0],
			TableID: tables[tableIndex].ID,
		}
		players = players[1:]

		if len(players) > 0 {
			match.Player2 = players[0]
			players = players[1:]
		} else {
			match.Winner = match.Player1
			if tournament.Stages[nextStage] == nil {
				tournament.Stages[nextStage] = make(map[string]bool)
			}
			tournament.Stages[nextStage][match.Player1] = true
		}

		round.Matches = append(round.Matches, match)
		tableIndex = (tableIndex + 1) % len(tables)
	}

	return round
}

// Updates the result of a match
func updateMatchResult(db *Database, matchID string, winnerName string) error {
	tournament := getCurrentTournament(db)
	if tournament == nil {
		return fmt.Errorf("no active tournament")
	}

	for j, round := range tournament.Rounds {
		for k, match := range round.Matches {
			if match.ID == matchID {
				if match.Player1 != winnerName && match.Player2 != winnerName {
					return fmt.Errorf("le gagnant doit être l'un des joueurs du match: %s ou %s", match.Player1, match.Player2)
				}

				db.Tournaments[len(db.Tournaments)-1].Rounds[j].Matches[k].Winner = winnerName
				return updateTournament(db)
			}
		}
	}
	return fmt.Errorf("match non trouvé")
}

// Updates the tournament status based on the current round
func updateTournament(db *Database) error {
	tournament := getCurrentTournament(db)
	if tournament == nil {
		return fmt.Errorf("no active tournament")
	}

	currentRound := tournament.Rounds[tournament.CurrentRound]
	allMatchesComplete := true

	for _, match := range currentRound.Matches {
		if match.Winner == "" {
			allMatchesComplete = false
			break
		}
		advanceToNextStage(tournament, match.Winner)
	}

	if allMatchesComplete {
		nextStage := fmt.Sprintf("stage%d", tournament.CurrentRound+2)
		if len(tournament.Stages[nextStage]) <= 1 {
			tournament.Status = TournamentStatusComplete
		} else {
			nextRound := createRound(tournament, db.Tables)
			tournament.Rounds = append(tournament.Rounds, nextRound)
			tournament.CurrentRound++
		}
	}

	return saveDatabase(*db)
}

// Creates a new tournament with the given player names
func createTournament(playerNames []string) *Tournament {
	tournament := &Tournament{
		ID:           fmt.Sprintf("T%d", len(playerNames)),
		CurrentRound: 0,
		Status:       TournamentStatusPending,
		Players:      playerNames,
		Stages:       make(map[string]map[string]bool),
	}

	for _, playerNames := range playerNames {
		tournament.addPlayerToStage("stage1", playerNames)
	}

	firstRound := Round{
		Matches: createMatches(playerNames),
	}

	tournament.Rounds = append(tournament.Rounds, firstRound)
	tournament.Status = TournamentStatusOngoing

	return tournament
}

// Adds a player to the specified stage
func (t *Tournament) addPlayerToStage(stageName string, playerName string) {
	if t.Stages[stageName] == nil {
		t.Stages[stageName] = make(map[string]bool)
	}
	t.Stages[stageName][playerName] = true
}

// Advances the winner of a match to the next stage
func advanceToNextStage(tournament *Tournament, winner string) {
	nextStage := fmt.Sprintf("stage%d", tournament.CurrentRound+2)
	if tournament.Stages[nextStage] == nil {
		tournament.Stages[nextStage] = make(map[string]bool)
	}
	tournament.Stages[nextStage][winner] = true
}

/* Utility functions */
// Returns the nearest even number
func nearestEvenNumber(n int) int {
	return n - (n % 2)
}

// Sends an embed message to the specified channel
func sendEmbed(s *discordgo.Session, channelID, title, description string, color int) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
	}
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Erreur lors de l'envoi de l'embed: %v", err)
	}

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
		if len(args) == 0 || args[0] != prefix {
			return
		}

		if len(args) < 2 {
			sendEmbed(s, m.ChannelID, "Erreur", "Commande incomplète. Utilisez !smashbot help pour voir les commandes disponibles.", 0xFF0000)
			return
		}

		if args[1] == "hello" {
			if _, err := s.ChannelMessageSend(m.ChannelID, "world!"); err != nil {
				log.Printf("Erreur lors de l'envoi du message: %v", err)
			}
		}

		db, err := loadDatabase()
		if err != nil {
			sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du chargement de la base de données", 0xFF0000)
			return
		}

		switch {
		//pour ajouter un joueur
		case len(args) >= 3 && args[1] == "add" && args[2] == "player":
			if len(args) < 4 {
				sendEmbed(s, m.ChannelID, "Erreur", "Usage: !smashbot help", 0xFF0000)
				return
			}

			newPlayer := Player{
				ID:       uuid.New().String(),
				Username: strings.Join(args[3:], " "),
			}

			err = addPlayer(db, newPlayer)
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
			err = removePlayer(db, username)
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
			err = addTable(db, numTables)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de l'ajout des tables : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", fmt.Sprintf("%d tables ajoutées avec succès!", numTables), 0x00FF00)

		//pour démarrer un tournoi
		case len(args) == 3 && args[1] == "tournament" && args[2] == "start":
			err := startTournament(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du démarrage du tournoi : "+err.Error(), 0xFF0000)
				return
			}

			tournament := getCurrentTournament(db)
			if tournament == nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de la récupération du tournoi", 0xFF0000)
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

		//pour mettre à jour le résultat d'un match
		case len(args) == 5 && args[1] == "match" && args[2] == "result":
			matchID := args[3]
			winnerName := args[4]
			err := updateMatchResult(db, matchID, winnerName)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors de la mise à jour du résultat : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "Résultat du match mis à jour avec succès!", 0x00FF00)

		//pour afficher l'état du tournoi
		case len(args) == 3 && args[1] == "tournament" && args[2] == "status":
			status := getTournamentStatus(*db)
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
			err = removeTables(db, numTables)
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
