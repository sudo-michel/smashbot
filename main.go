package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"log"
	"math"
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
	Classe   string `json:"classe"`
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
	IsFirstRound bool                       `json:"is_first_round"`
}

type Match struct {
	ID              string   `json:"id"`
	Players         []string `json:"players"`
	Player1         string   `json:"player1"`
	Player2         string   `json:"player2"`
	Winner          string   `json:"winner"`
	TableID         string   `json:"table_id"`
	Classe          string   `json:"classe"`
	NextmatchID     string   `json:"next_match_id"`
	WaitingForMatch string   `json:"waiting_for_match"`
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

// Lists all players in the database
func listTables(db *Database) string {
	if len(db.Tables) == 0 {
		return "No players"
	}
	var playersList strings.Builder
	for i, Tables := range db.Tables {
		playersList.WriteString(fmt.Sprintf("%d. %s\n", i+1, Tables.ID))
	}
	return playersList.String()
}

// Adds new table to database
func addTable(db *Database, numTables int) error {
	for i := 0; i < numTables; i++ {
		newTable := Table{
			ID:        strconv.Itoa(i),
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
		ID:           strconv.Itoa(len(db.Tournaments) + 1),
		CurrentRound: 0,
		Status:       TournamentStatusPending,
		Players:      make([]string, 0),
		Stages:       make(map[string]map[string]bool),
		IsFirstRound: true,
	}

	players := make([]Player, len(db.Players))
	copy(players, db.Players)
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	firstRoundMatches := firstRound(players, db.Tables)

	tournament.Rounds = append(tournament.Rounds, Round{
		Matches: firstRoundMatches,
	})
	tournament.Status = TournamentStatusOngoing

	for _, p := range db.Players {
		tournament.Players = append(tournament.Players, p.Username)
	}

	db.Tournaments = append(db.Tournaments, tournament)
	return saveDatabase(*db)
}

func nextRound(db *Database) error {
	tournament := getCurrentTournament(db)
	if tournament == nil {
		return fmt.Errorf("no active tournament")
	}
	if tournament.Status != TournamentStatusOngoing {
		return fmt.Errorf("le tournoi n'est pas en cours")
	}

	currentRound := tournament.Rounds[tournament.CurrentRound]
	winners := []string{}
	for _, match := range currentRound.Matches {
		if match.Winner == "" {
			return fmt.Errorf("tous les matches du tour actuel ne sont pas terminés")

		}
		winners = append(winners, match.Winner)
	}

	if len(winners) == 1 {
		tournament.Status = TournamentStatusComplete
		return saveDatabase(*db)
	}

	var nextMatches []Match
	if tournament.IsFirstRound {
		tournament.IsFirstRound = false
		nextMatches = manageMatches(winners, db.Tables)
	} else {
		nextMatches = manageMatches(winners, db.Tables)
	}

	tournament.Rounds = append(tournament.Rounds, Round{
		Matches: nextMatches,
	})
	tournament.CurrentRound++
	return saveDatabase(*db)
}

// Creates matches for the first round of the tournament
func firstRound(players []Player, tables []Table) []Match {
	totalPlayers := len(players)
	powerofTwo := LargestPowerOfTwo(totalPlayers)
	halfpowerofTwo := powerofTwo / 2

	log.Print("Total Players : ", totalPlayers)
	log.Print("Power of Two : ", powerofTwo)
	log.Print("Half Power of Two : ", halfpowerofTwo)

	currentPlayerIndex := 0
	matches := []Match{}
	matchCounter := 1
	tableIndex := 0

	for i := 0; i < halfpowerofTwo; i += 2 {
		match := Match{
			ID:      strconv.Itoa(matchCounter),
			Player1: players[currentPlayerIndex].Username,
			Player2: players[currentPlayerIndex+1].Username,
			Winner:  "",
			Classe:  "A",
			TableID: tables[tableIndex%len(tables)].ID,
		}
		players[currentPlayerIndex].Classe = "A"
		players[currentPlayerIndex+1].Classe = "A"
		matches = append(matches, match)
		matchCounter++
		tableIndex++
		currentPlayerIndex += 2
	}

	remainingPlayers := totalPlayers - halfpowerofTwo
	log.Print("Remaining Players : ", remainingPlayers)
	pairsNeeded := remainingPlayers / 2
	log.Print("Pairs Needed : ", pairsNeeded)
	for i := 0; i < pairsNeeded; i += 2 {
		if currentPlayerIndex+1 >= totalPlayers {
			break
		}
		match := Match{
			ID:      strconv.Itoa(matchCounter),
			Player1: players[currentPlayerIndex].Username,
			Player2: players[currentPlayerIndex+1].Username,
			Winner:  "",
			Classe:  "B",
			TableID: tables[tableIndex%len(tables)].ID,
		}
		players[currentPlayerIndex].Classe = "B"
		players[currentPlayerIndex+1].Classe = "B"
		matches = append(matches, match)
		tableIndex++
		matchCounter++
		currentPlayerIndex += 2
	}

	remainingForTriples := totalPlayers - currentPlayerIndex
	log.Print("Remaining Players for triples : ", remainingForTriples)

	for remainingForTriples >= 3 {
		tripleGroup := []Player{
			players[currentPlayerIndex],
			players[currentPlayerIndex+1],
			players[currentPlayerIndex+2],
		}

		match1 := Match{
			ID:          strconv.Itoa(matchCounter),
			Player1:     tripleGroup[0].Username,
			Player2:     tripleGroup[1].Username,
			Winner:      "",
			Classe:      "C",
			NextmatchID: fmt.Sprintf("M%d", len(matches)+2),
			TableID:     tables[tableIndex%len(tables)].ID,
		}

		match2 := Match{
			ID:              strconv.Itoa(matchCounter),
			Player1:         tripleGroup[2].Username,
			Player2:         "",
			Winner:          "",
			Classe:          "C",
			TableID:         tables[tableIndex%len(tables)].ID,
			WaitingForMatch: match1.ID,
		}
		matchCounter++
		tableIndex++

		matches = append(matches, match1, match2)
		currentPlayerIndex += 3
		remainingForTriples -= 3
	}

	remainingPlayers = totalPlayers - currentPlayerIndex

	if remainingPlayers > 0 {
		log.Print("Remaining Players : ", remainingPlayers)
		if remainingPlayers == 2 {
			match := Match{
				ID:      strconv.Itoa(matchCounter),
				Player1: players[currentPlayerIndex].Username,
				Player2: players[currentPlayerIndex+1].Username,
				Winner:  "",
				Classe:  "D",
				TableID: tables[tableIndex%len(tables)].ID,
			}
			matches = append(matches, match)
			matchCounter++

		} else if remainingPlayers == 1 {
			match := Match{
				ID:      strconv.Itoa(matchCounter),
				Player1: players[currentPlayerIndex].Username,
				Player2: "",
				Winner:  players[currentPlayerIndex].Username,
				Classe:  "D",
				TableID: tables[tableIndex%len(tables)].ID,
			}
			matches = append(matches, match)
			matchCounter++
		}
	}

	return matches
}

func manageMatches(winners []string, tables []Table) []Match {
	matches := []Match{}
	matchCounter := 1

	for i := 0; i < len(winners); i += 2 {
		if i+1 < len(winners) {
			match := Match{
				ID:      strconv.Itoa(matchCounter),
				Player1: winners[i],
				Player2: winners[i+1],
				Winner:  "",
				TableID: tables[i/2%len(tables)].ID,
			}
			matches = append(matches, match)
			matchCounter++
		}
	}

	return matches
}

// Updates the result of a match
func updateMatchResult(db *Database, matchID string, winnerName string) error {
	tournament := getCurrentTournament(db)
	if tournament == nil {
		return fmt.Errorf("no active tournament")
	}

	var updateMatch *Match
	matchFound := false

	for j, round := range tournament.Rounds {
		for k, match := range round.Matches {
			if match.ID == matchID {
				if match.Player1 != winnerName && match.Player2 != winnerName {
					return fmt.Errorf("le gagnant doit être l'un des joueurs du match: %s ou %s", match.Player1, match.Player2)
				}
				tournament.Rounds[j].Matches[k].Winner = winnerName
				updateMatch = &tournament.Rounds[j].Matches[k]
				matchFound = true
				break

			}
		}
		if matchFound {
			break
		}
	}

	if updateMatch == nil {
		return fmt.Errorf("match non trouvé")

	}
	if updateMatch.NextmatchID != "" {
		nextMatchFound := false
		for j, round := range tournament.Rounds {
			for k, match := range round.Matches {
				if match.ID == updateMatch.NextmatchID {
					if match.Player1 == "" {
						tournament.Rounds[j].Matches[k].Player1 = winnerName
					} else {
						tournament.Rounds[j].Matches[k].Player2 = winnerName
					}
					nextMatchFound = true
					break
				}
			}
			if nextMatchFound {
				break
			}
		}
	}

	return saveDatabase(*db)

}

// Adds a player to the specified stage
func (t *Tournament) addPlayerToStage(stageName string, playerName string) {
	if t.Stages[stageName] == nil {
		t.Stages[stageName] = make(map[string]bool)
	}
	t.Stages[stageName][playerName] = true
}

/* Utility functions */

// Returns the nearest even number
func LargestPowerOfTwo(n int) int {
	if n <= 0 {
		return 0
	}

	power := int(math.Floor(math.Log2(float64(n))))

	return int(math.Pow(2, float64(power)))

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

func clearTournaments(db *Database) error {
	db.Tournaments = []Tournament{}
	return saveDatabase(*db)
}
func clearPlayers(db *Database) error {
	db.Players = []Player{}
	return saveDatabase(*db)
}
func clearTables(db *Database) error {
	db.Tables = []Table{}
	return saveDatabase(*db)
}
func clearDatabase(db *Database) error {
	db.Players = []Player{}
	db.Tables = []Table{}
	db.Tournaments = []Tournament{}
	return saveDatabase(*db)
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

		//pour lister les tables
		case len(args) == 3 && args[1] == "list" && args[2] == "table":
			tableList := listTables(db)
			sendEmbed(s, m.ChannelID, "Liste des tables", tableList, 0x00FF00)

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

		/*Clear database */

		case len(args) == 3 && args[1] == "clear" && args[2] == "tournament":
			err := clearTournaments(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du nettoyage des tournois : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "Tous les tournois ont été supprimés!", 0x00FF00)

		case len(args) == 3 && args[1] == "clear" && args[2] == "player":
			err := clearPlayers(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du nettoyage des joueurs : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "Tous les joueurs ont été supprimés!", 0x00FF00)

		case len(args) == 3 && args[1] == "clear" && args[2] == "table":
			err := clearTables(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du nettoyage des tables : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "Toutes les tables ont été supprimées!", 0x00FF00)

		case len(args) == 3 && args[1] == "clear" && args[2] == "ALL":
			err := clearDatabase(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du nettoyage de la base de données : "+err.Error(), 0xFF0000)
				return
			}
			sendEmbed(s, m.ChannelID, "Succès", "La base de données a été entièrement nettoyée!", 0x00FF00)

		//pour passer au tour suivant
		case len(args) == 3 && args[1] == "tournament" && args[2] == "next":
			err := nextRound(db)
			if err != nil {
				sendEmbed(s, m.ChannelID, "Erreur", "Erreur lors du passage au tour suivant : "+err.Error(), 0xFF0000)
				return
			}

			tournament := getCurrentTournament(db)
			if tournament.Status == TournamentStatusComplete {
				// Trouver le gagnant final
				lastRound := tournament.Rounds[len(tournament.Rounds)-1]
				winner := lastRound.Matches[0].Winner
				sendEmbed(s, m.ChannelID, "Tournoi terminé!", fmt.Sprintf("Le tournoi est terminé! Le gagnant est : %s", winner), 0x00FF00)
			} else {
				// Afficher les nouveaux matches
				var matchesInfo strings.Builder
				currentRound := tournament.Rounds[tournament.CurrentRound]
				matchesInfo.WriteString(fmt.Sprintf("Tour %d :\n", tournament.CurrentRound+1))
				for _, match := range currentRound.Matches {
					if match.Player2 != "" {
						matchesInfo.WriteString(fmt.Sprintf("Match ID: %s - %s vs %s (Table: %s)\n",
							match.ID, match.Player1, match.Player2, match.TableID))
					} else {
						matchesInfo.WriteString(fmt.Sprintf("Match ID: %s - %s passe automatiquement (Table: %s)\n",
							match.ID, match.Player1, match.TableID))
					}
				}
				sendEmbed(s, m.ChannelID, "Nouveau tour commencé", matchesInfo.String(), 0x00FF00)
			}

		//pour afficher l'aide
		case len(args) == 2 && args[1] == "help":
			helpMessage := `Commandes disponibles:
Gestion des joueurs:
  !smashbot add player <username>    : Ajoute un nouveau joueur
  !smashbot remove player <username> : Supprime un joueur existant
  !smashbot list player              : Affiche la liste des joueurs
  !smashbot clear player             : Supprime tous les joueurs

Gestion des tables:
  !smashbot add tables <number>    : Ajoute un certain nombre de tables
  !smashbot remove tables <number> : Supprime un certain nombre de tables
  !smashbot clear table            : Supprime toutes les tables

Gestion des tournois:
  !smashbot tournament start : Démarre un nouveau tournoi
  !smashbot tournament next : Passe au tour suivant
  !smashbot match result <match_id> <winner_name> : Enregistre le résultat d'un match
  !smashbot clear tournament : Supprime tous les tournois

Nettoyage global:
  !smashbot clear ALL: Vide entièrement la base de données

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
