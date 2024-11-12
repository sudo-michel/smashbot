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

	// Vérifier tous les matchs d'abord
	for _, match := range currentRound.Matches {
		if match.Player2 == "" {
			if match.Winner == "" {
				match.Winner = match.Player1
			}
			winners = append(winners, match.Winner)
		} else {
			if match.Winner == "" {
				return fmt.Errorf("le match %s n'est pas terminé", match.ID)
			}
			winners = append(winners, match.Winner)
		}
	}

	if len(winners) == 1 {
		tournament.Status = TournamentStatusComplete
		return saveDatabase(*db)
	}

	nextRoundNumber := tournament.CurrentRound + 2
	var nextMatches []Match

	if tournament.IsFirstRound {
		tournament.IsFirstRound = false
		nextMatches = manageMatches(winners, db.Tables, nextRoundNumber)
	} else {
		nextMatches = manageMatches(winners, db.Tables, nextRoundNumber)
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
	targetSize := LargestPowerOfTwo(totalPlayers) / 2
	matchesNeeded := targetSize

	log.Printf("Total Players: %d", totalPlayers)
	log.Printf("Target Size for next round: %d", targetSize)
	log.Printf("Matches needed: %d", matchesNeeded)

	matches := []Match{}
	currentPlayerIndex := 0
	//remainingPlayers := totalPlayers
	tableIndex := 0

	playerInMatches := (totalPlayers - targetSize) * 2
	if playerInMatches < 0 {
		playerInMatches = totalPlayers - (totalPlayers % 2)
	}
	byePlayers := totalPlayers - playerInMatches

	log.Print("Players in matches : ", playerInMatches)
	log.Print("Bye Players : ", byePlayers)

	matchCounter := 1
	matchesCreated := 0

	for matchesCreated < playerInMatches/2 {
		if currentPlayerIndex+1 >= totalPlayers {
			break
		}
		match := Match{
			ID:      fmt.Sprintf("R1M%d", matchCounter),
			Player1: players[currentPlayerIndex].Username,
			Player2: players[currentPlayerIndex+1].Username,
			Winner:  "",
			TableID: tables[tableIndex%len(tables)].ID,
		}
		matches = append(matches, match)
		currentPlayerIndex += 2
		matchCounter++
		matchesCreated++
		tableIndex++

	}

	for i := 0; i < byePlayers; i++ {
		if currentPlayerIndex >= len(players) {
			break
		}

		match := Match{
			ID:      fmt.Sprintf("R1M%d", matchCounter),
			Player1: players[currentPlayerIndex].Username,
			Player2: "",
			Winner:  players[currentPlayerIndex].Username, // Le joueur passe automatiquement
			TableID: "",                                   // Pas besoin de table pour un bye
		}

		matches = append(matches, match)
		currentPlayerIndex++
		matchCounter++
	}

	return matches
}

func manageMatches(winners []string, tables []Table, roundNumber int) []Match {
	matches := []Match{}
	matchCounter := 1

	for i := 0; i < len(winners); i += 2 {
		if i+1 < len(winners) {
			match := Match{
				ID:      fmt.Sprintf("R%dM%d", roundNumber, matchCounter),
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
					tournament.Rounds[j].Matches[k].Player2 = winnerName
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
		return 1
	}
	power := math.Ceil(math.Log2(float64(n)))
	return int(math.Pow(2, power))

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

func registerCommands(s *discordgo.Session) {
	log.Print("Registering commands...")

	if s == nil || s.State == nil || s.State.User == nil {
		log.Print("Session or user not found")
		return
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "smashbot",
			Description: "Commandes principales du bot Smashbot",
			Type:        discordgo.ApplicationCommandType(1),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Ajouter un joueur ou des tables",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "player",
							Description: "Ajouter un nouveau joueur",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "username",
									Description: "Nom du joueur",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
						{
							Name:        "tables",
							Description: "Ajouter des tables",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "number",
									Description: "Nombre de tables à ajouter",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},

				{
					Name:        "remove",
					Description: "Supprimer un joueur ou des tables",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "player",
							Description: "Nom du joueur à supprimer",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "username",
									Description: "Nom du joueur",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
						{
							Name:        "tables",
							Description: "Supprimer des tables",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "number",
									Description: "Nombre de tables à supprimer",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},
				{
					Name:        "list",
					Description: "Lister les joueurs ou les tables",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "type",
							Description: "Type d'élément à lister (player/table)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Joueurs",
									Value: "player",
								},
								{
									Name:  "Tables",
									Value: "table",
								},
							},
						},
					},
				},
				{
					Name:        "tournament",
					Description: "Gérer le tournoi",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "action",
							Description: "Action à effectuer (start/next/status)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "start",
									Value: "start",
								},
								{
									Name:  "next",
									Value: "next",
								},
								{
									Name:  "status",
									Value: "status",
								},
							},
						},
					},
				},
				{
					Name:        "match",
					Description: "Gérer les résultats des matchs",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "match_id",
							Description: "ID du match",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "winner",
							Description: "Nom du gagnant",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "clear",
					Description: "Nettoyer la base de données",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "type",
							Description: "Type d'élément à nettoyer (tournament/player/table/ALL)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "tournament",
									Value: "tournament",
								},
								{
									Name:  "player",
									Value: "player",
								},
								{
									Name:  "table",
									Value: "table",
								},
								{
									Name:  "ALL",
									Value: "ALL",
								},
							},
						},
					},
				},
			},
		},
	}

	// Enregistrer les commandes
	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Erreur lors de la création de la commande %v: %v", cmd.Name, err)
		}
	}
	log.Println("Commands registered successfully!")
}

func sendInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, color int) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
				},
			},
		},
	})
}

func handleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	if data.Name != "smashbot" {
		return
	}

	// Charger la base de données
	db, err := loadDatabase()
	if err != nil {
		sendInteractionResponse(s, i, "Erreur", "Erreur lors du chargement de la base de données", 0xFF0000)
		return
	}

	// Obtenir la sous-commande et ses options
	if len(data.Options) == 0 {
		sendInteractionResponse(s, i, "Erreur", "Commande invalide", 0xFF0000)
		return
	}

	groupCmd := data.Options[0]

	switch groupCmd.Name {
	case "add":
		if len(groupCmd.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Options manquantes", 0xFF0000)
			return
		}

		subCmd := groupCmd.Options[0]
		switch subCmd.Name {
		case "player":
			if len(subCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Nom du joueur manquant", 0xFF0000)
				return
			}
			username := subCmd.Options[0].StringValue()
			newPlayer := Player{
				ID:       uuid.New().String(),
				Username: username,
			}
			err = addPlayer(db, newPlayer)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors de l'ajout du joueur: "+err.Error(), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Succès", fmt.Sprintf("Joueur %s ajouté avec succès!", newPlayer.Username), 0x00FF00)

		case "tables":
			if len(subCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Nombre de tables manquant", 0xFF0000)
				return
			}
			numTables := int(subCmd.Options[0].IntValue())
			err = addTable(db, numTables)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors de l'ajout des tables: "+err.Error(), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Succès", fmt.Sprintf("%d tables ajoutées avec succès!", numTables), 0x00FF00)
		}

	case "remove":
		if len(groupCmd.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Options manquantes", 0xFF0000)
			return
		}

		subCmd := groupCmd.Options[0]
		switch subCmd.Name {
		case "player":
			if len(subCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Nom du joueur manquant", 0xFF0000)
				return
			}
			username := subCmd.Options[0].StringValue()
			err = removePlayer(db, username)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors de la suppression du joueur: "+err.Error(), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Succès", fmt.Sprintf("Joueur %s supprimé avec succès!", username), 0x00FF00)

		case "tables":
			if len(subCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Nombre de tables manquant", 0xFF0000)
				return
			}
			numTables := int(subCmd.Options[0].IntValue())
			err = removeTables(db, numTables)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors de la suppression des tables: "+err.Error(), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Succès", fmt.Sprintf("%d tables supprimées avec succès!", numTables), 0x00FF00)
		}

	case "list":
		if len(groupCmd.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Type de liste manquant", 0xFF0000)
			return
		}
		listType := groupCmd.Options[0].StringValue()
		var list string
		switch listType {
		case "player":
			list = listPlayers(db)
		case "table":
			list = listTables(db)
		}
		sendInteractionResponse(s, i, fmt.Sprintf("Liste des %s", listType), list, 0x00FF00)

	case "tournament":
		if len(groupCmd.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Action manquante", 0xFF0000)
			return
		}
		action := groupCmd.Options[0].StringValue()
		switch action {
		case "start":
			err := startTournament(db)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors du démarrage du tournoi: "+err.Error(), 0xFF0000)
				return
			}
			tournament := getCurrentTournament(db)
			var matchesInfo strings.Builder
			matchesInfo.WriteString(fmt.Sprintf("Tournoi ID: %s\n\n", tournament.ID))
			matchesInfo.WriteString("Liste des joueurs:\n")
			for i, player := range tournament.Players {
				matchesInfo.WriteString(fmt.Sprintf("%d. %s\n", i+1, player))
			}
			matchesInfo.WriteString("\nMatches du premier tour:\n")
			for _, match := range tournament.Rounds[0].Matches {
				matchesInfo.WriteString(fmt.Sprintf("Match %s: %s vs %s (Table: %s)\n",
					match.ID, match.Player1, match.Player2, match.TableID))
			}
			sendInteractionResponse(s, i, "Tournoi démarré", matchesInfo.String(), 0x00FF00)

		case "status":
			status := getTournamentStatus(*db)
			sendInteractionResponse(s, i, "État du tournoi", status, 0x00FF00)

		case "next":
			err := nextRound(db)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Erreur lors du passage au tour suivant: "+err.Error(), 0xFF0000)
				return
			}

			tournament := getCurrentTournament(db)
			if tournament.Status == TournamentStatusComplete {
				lastRound := tournament.Rounds[len(tournament.Rounds)-1]
				winner := lastRound.Matches[0].Winner
				sendInteractionResponse(s, i, "Tournoi terminé!", fmt.Sprintf("Le tournoi est terminé! Le gagnant est : %s", winner), 0x00FF00)
				return
			}

			currentRound := tournament.Rounds[tournament.CurrentRound]
			var matchesInfo strings.Builder
			matchesInfo.WriteString(fmt.Sprintf("Tour %d:\n\n", tournament.CurrentRound+1))

			for _, match := range currentRound.Matches {
				if match.Player2 == "" {
					matchesInfo.WriteString(fmt.Sprintf("Match %s: %s passe automatiquement\n",
						match.ID, match.Player1))
				} else {
					matchesInfo.WriteString(fmt.Sprintf("Match %s: %s vs %s (Table: %s)\n",
						match.ID, match.Player1, match.Player2, match.TableID))
				}
			}

			sendInteractionResponse(s, i, "Nouveau tour commencé", matchesInfo.String(), 0x00FF00)
		}

	case "match":
		if len(groupCmd.Options) < 2 {
			sendInteractionResponse(s, i, "Erreur", "ID du match et gagnant requis", 0xFF0000)
			return
		}
		matchID := groupCmd.Options[0].StringValue()
		winnerName := groupCmd.Options[1].StringValue()
		err := updateMatchResult(db, matchID, winnerName)
		if err != nil {
			sendInteractionResponse(s, i, "Erreur", "Erreur lors de la mise à jour du résultat: "+err.Error(), 0xFF0000)
			return
		}
		sendInteractionResponse(s, i, "Succès", "Résultat du match mis à jour avec succès!", 0x00FF00)

	case "clear":
		if len(groupCmd.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Type de nettoyage requis", 0xFF0000)
			return
		}
		clearType := groupCmd.Options[0].StringValue()
		var err error
		switch clearType {
		case "tournament":
			err = clearTournaments(db)
		case "player":
			err = clearPlayers(db)
		case "table":
			err = clearTables(db)
		case "ALL":
			err = clearDatabase(db)
		}
		if err != nil {
			sendInteractionResponse(s, i, "Erreur", fmt.Sprintf("Erreur lors du nettoyage: %v", err), 0xFF0000)
			return
		}
		sendInteractionResponse(s, i, "Succès", "Nettoyage effectué avec succès!", 0x00FF00)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erreur lors du chargement du fichier .env")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("Le token du bot n'est pas défini dans le fichier .env")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	// Ajouter le handler pour les commandes slash
	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Bot is ready! Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		// Enregistrer les commandes slash une fois que le bot est prêt
		registerCommands(s)
	})

	sess.AddHandler(handleCommands)

	// Définir les intents nécessaires
	sess.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildIntegrations

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	log.Print("Bot is running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
