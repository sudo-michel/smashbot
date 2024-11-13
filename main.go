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
	ID           string           `json:"id"`
	Matches      []Match          `json:"matches"`
	Rounds       []Round          `json:"rounds"`
	Players      []string         `json:"player_ids"`
	Status       TournamentStatus `json:"status"`
	CurrentRound int              `json:"current_round"`
	IsFirstRound bool             `json:"is_first_round"`
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

var securityCodes = map[string]int{
	"tournament": 0,
	"player":     0,
	"table":      0,
	"ALL":        0,
}

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
	// Check if player already exists
	for _, p := range db.Players {
		if p.Username == player.Username {
			return fmt.Errorf("player already exists")
		}
	}
	db.Players = append(db.Players, player)
	log.Print("Player added successfully")
	return saveDatabase(*db)
}

// Removes player from database
func removePlayer(db *Database, username string) error {
	for i, p := range db.Players {
		if p.Username == username {

			db.Players = append(db.Players[:i], db.Players[i+1:]...)
			return saveDatabase(*db)
		}
	}
	log.Print("Player removed successfully")
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
	log.Print("List of player sent successfully")
	return playersList.String()
}

// Lists all players in the database
func listTables(db *Database) string {
	if len(db.Tables) == 0 {
		return "No tables"
	}
	var playersList strings.Builder
	for i, Tables := range db.Tables {
		playersList.WriteString(fmt.Sprintf("%d. %s\n", i+1, Tables.ID))
	}
	log.Print("List of tables sent successfully")
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
	log.Print("Table added successfully")
	return saveDatabase(*db)
}

// Removes table from database
func removeTables(db *Database, numTables int) error {
	if numTables > len(db.Tables) {
		return fmt.Errorf("not enough tables to delete")
	}
	db.Tables = db.Tables[:len(db.Tables)-numTables]
	log.Print("Table removed successfully")
	return saveDatabase(*db)
}

// Returns the current tournament
func getCurrentTournament(db *Database) *Tournament {
	if len(db.Tournaments) == 0 {
		return nil
	}
	log.Print("Current tournament found")
	return &db.Tournaments[len(db.Tournaments)-1]
}

// Updates the database with the current tournament
func getTournamentStatus(db Database) string {
	tournament := getCurrentTournament(&db)
	if tournament == nil {
		return "No tournaments in progress."
	}

	if tournament.Status == TournamentStatusComplete {
		return "No tournaments in progress. The last tournament is over."
	}

	status := fmt.Sprintf("Tournament status (ID: %s):\n", tournament.ID)
	status += fmt.Sprintf("Statut: %s\n", tournament.Status)
	status += fmt.Sprintf("Current tour: %d\n\n", tournament.CurrentRound+1)

	currentRound := tournament.Rounds[tournament.CurrentRound]
	status += "Matches in progress:\n"
	for _, match := range currentRound.Matches {
		status += fmt.Sprintf("- Matches %s: %s vs %s", match.ID, match.Player1, match.Player2)
		if match.Winner != "" {
			status += fmt.Sprintf(" (Winner: %s)", match.Winner)
		}
		status += "\n"
	}
	log.Print("Tournament status sent successfully")
	return status
}

// Starts a new tournament
func startTournament(db *Database) error {
	if len(db.Players) < 2 {
		return fmt.Errorf("not enough players to start a tournament. Minimum 2 players required")
	}

	if len(db.Tables) == 0 {
		return fmt.Errorf("no table available")

	}

	tournament := Tournament{
		ID:           strconv.Itoa(len(db.Tournaments) + 1),
		CurrentRound: 0,
		Status:       TournamentStatusPending,
		Players:      make([]string, 0),

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
	log.Print("Tournament started successfully")
	return saveDatabase(*db)
}

func nextRound(db *Database) error {
	tournament := getCurrentTournament(db)
	if tournament == nil {
		return fmt.Errorf("no active tournament")
	}
	if tournament.Status != TournamentStatusOngoing {
		return fmt.Errorf("the tournament is not in progress")
	}

	currentRound := tournament.Rounds[tournament.CurrentRound]
	winners := []string{}

	for _, match := range currentRound.Matches {
		if match.Player2 == "" {
			if match.Winner == "" {
				match.Winner = match.Player1
			}
			winners = append(winners, match.Winner)
		} else {
			if match.Winner == "" {
				return fmt.Errorf("the %s match is not over", match.ID)
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
	log.Print("Next round started successfully")
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

	var matches []Match
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
			Winner:  players[currentPlayerIndex].Username, //The player automatically passes
			TableID: "",                                   // No table needed for a bye
		}

		matches = append(matches, match)
		currentPlayerIndex++
		matchCounter++
	}
	log.Print("First round matches created successfully")
	return matches
}

// Creates matches for the next round of the tournament
func manageMatches(winners []string, tables []Table, roundNumber int) []Match {
	var matches []Match
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
	log.Print("Next round matches created successfully")
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
					return fmt.Errorf("the winner must be one of the players in the match: %s ou %s", match.Player1, match.Player2)
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
		return fmt.Errorf("match not found")

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
	log.Print("Match updated successfully")
	return saveDatabase(*db)
}

// Returns the nearest even number
func LargestPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	power := math.Ceil(math.Log2(float64(n)))
	return int(math.Pow(2, power))
}

func verifySecurityCode(clearType string, userCode string) error {
	inputCode, err := strconv.Atoi(userCode)
	if err != nil {
		return fmt.Errorf("invalid security code : %v", err)
	}

	if inputCode != securityCodes[clearType] {
		return fmt.Errorf("incorrect security code")
	}
	log.Print("Security code verified successfully")
	return nil
}

func generateSecurityCode(clearType string) int {
	code := rand.Int() % 100000
	securityCodes[clearType] = code
	log.Print("Security code generated: ", code)
	return code
}

func clearTournament(db *Database) error {
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
			Name:        "activedevbadge",
			Description: "Command to obtain the Active Developer badge",
			Type:        discordgo.ApplicationCommandType(1),
		},
		{
			Name:        "smashbot",
			Description: "Main Smashbot commands",
			Type:        discordgo.ApplicationCommandType(1),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a player or tables",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "player",
							Description: "Add new player",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "username",
									Description: "Name of the player",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
						{
							Name:        "tables",
							Description: "Add tables",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "number",
									Description: "Number of tables to add",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},

				{
					Name:        "remove",
					Description: "Delete a player or tables",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "player",
							Description: "Name of the player to remove",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "username",
									Description: "Name of the player",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
						{
							Name:        "tables",
							Description: "delete tables",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "number",
									Description: "Number of tables to delete",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},
				{
					Name:        "list",
					Description: "List of player or tables",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "type",
							Description: "Type of item to list (player/table)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Players",
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
					Description: "Manage tournaments",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "action",
							Description: "Action to be taken (start/next/status)",
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
					Description: "Manage match results",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "match_id",
							Description: "ID of the match",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "winner",
							Description: "Name of the winner",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "clear",
					Description: "clear the database",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "type",
							Description: "Type of element to be cleaned (tournament/player/table/ALL)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Tournament",
									Value: "tournament",
								},
								{
									Name:  "Players",
									Value: "player",
								},
								{
									Name:  "Tables",
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
				{
					Name:        "confirm-clear",
					Description: "Confirm tournament deletion with security code",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "code",
							Description: "Security code received",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
						{
							Name:        "type",
							Description: "Type d'élément à nettoyer (tournoi/joueur/table/tous)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Tournament",
									Value: "tournament",
								},
								{
									Name:  "Players",
									Value: "player",
								},
								{
									Name:  "Tables",
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
				{
					Name:        "help",
					Description: "Display all available commands",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	// Register commands
	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Order creation error %v: %v", cmd.Name, err)
		}
	}
	log.Println("Commands registered successfully!")
}

func sendInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, color int) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
	if err != nil {
		return
	}
}

// Main function to handle commands
func handleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()

	switch data.Name {
	case "activedevbadge":
		sendInteractionResponse(s, i, "Active Developer Badge",
			"This command helps you get your Active Developer badge. Visit https://discord.com/developers/active-developer to claim your badge.",
			0x00FF00)
		log.Print("Active Developer Badge sent successfully")
		return

	case "smashbot":
		// Load the database
		db, err := loadDatabase()
		if err != nil {
			sendInteractionResponse(s, i, "Erreur", "Error loading database", 0xFF0000)
			return
		}

		if len(data.Options) == 0 {
			sendInteractionResponse(s, i, "Erreur", "Commande invalide", 0xFF0000)
			return
		}

		groupCmd := data.Options[0]
		switch groupCmd.Name {
		case "add":
			if len(groupCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Missing options", 0xFF0000)
				return
			}

			subCmd := groupCmd.Options[0]
			switch subCmd.Name {
			case "player":
				if len(subCmd.Options) == 0 {
					sendInteractionResponse(s, i, "Erreur", "Name of the player missing", 0xFF0000)
					return
				}
				username := subCmd.Options[0].StringValue()
				newPlayer := Player{
					ID:       uuid.New().String(),
					Username: username,
				}
				err = addPlayer(db, newPlayer)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Error adding player: "+err.Error(), 0xFF0000)
					return
				}
				sendInteractionResponse(s, i, "Succès", fmt.Sprintf("Player %s added successfully!", newPlayer.Username), 0x00FF00)
				log.Print("Player added successfully")

			case "tables":
				if len(subCmd.Options) == 0 {
					sendInteractionResponse(s, i, "Erreur", "Number of tables missing", 0xFF0000)
					return
				}
				numTables := int(subCmd.Options[0].IntValue())
				err = addTable(db, numTables)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Error adding tables: "+err.Error(), 0xFF0000)
					return
				}
				sendInteractionResponse(s, i, "Succès", fmt.Sprintf("%d table successfully added!", numTables), 0x00FF00)
				log.Print("Tables added successfully")
			}

		case "remove":
			if len(groupCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Missing options", 0xFF0000)
				return
			}

			subCmd := groupCmd.Options[0]
			switch subCmd.Name {
			case "player":
				if len(subCmd.Options) == 0 {
					sendInteractionResponse(s, i, "Erreur", "Missing player name", 0xFF0000)
					return
				}
				username := subCmd.Options[0].StringValue()
				err = removePlayer(db, username)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Error when deleting player: "+err.Error(), 0xFF0000)
					return
				}
				sendInteractionResponse(s, i, "Succès", fmt.Sprintf("Player %s successfully deleted!", username), 0x00FF00)
				log.Print("Player removed successfully")

			case "tables":
				if len(subCmd.Options) == 0 {
					sendInteractionResponse(s, i, "Erreur", "Number of missing tables", 0xFF0000)
					return
				}
				numTables := int(subCmd.Options[0].IntValue())
				err = removeTables(db, numTables)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Error deleting tables: "+err.Error(), 0xFF0000)
					return
				}
				sendInteractionResponse(s, i, "Succès", fmt.Sprintf("%d tables successfully deleted !", numTables), 0x00FF00)
				log.Print("Tables removed successfully")
			}

		case "list":
			if len(groupCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Missing list type", 0xFF0000)
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
			sendInteractionResponse(s, i, fmt.Sprintf("List of %s", listType), list, 0x00FF00)
			log.Print("List sent successfully")

		case "tournament":
			if len(groupCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "(Action missing)", 0xFF0000)
				return
			}
			action := groupCmd.Options[0].StringValue()
			switch action {
			case "start":
				err := startTournament(db)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Tournament startup error : "+err.Error(), 0xFF0000)
					return
				}
				tournament := getCurrentTournament(db)
				var matchesInfo strings.Builder
				matchesInfo.WriteString(fmt.Sprintf("Tournoi ID: %s\n\n", tournament.ID))
				matchesInfo.WriteString("List of players:\n")
				for i, player := range tournament.Players {
					matchesInfo.WriteString(fmt.Sprintf("%d. %s\n", i+1, player))
				}
				matchesInfo.WriteString("\nFirst-round matches:\n")
				for _, match := range tournament.Rounds[0].Matches {
					matchesInfo.WriteString(fmt.Sprintf("Match %s: %s vs %s (Table: %s)\n",
						match.ID, match.Player1, match.Player2, match.TableID))
				}
				sendInteractionResponse(s, i, "Tournament started", matchesInfo.String(), 0x00FF00)
				log.Print("Tournament started successfully")

			case "status":
				status := getTournamentStatus(*db)
				sendInteractionResponse(s, i, "Tournament status", status, 0x00FF00)
				log.Print("Tournament status sent successfully")

			case "next":
				err := nextRound(db)
				if err != nil {
					sendInteractionResponse(s, i, "Erreur", "Error when moving on to the next lap: "+err.Error(), 0xFF0000)
					return
				}

				tournament := getCurrentTournament(db)
				if tournament.Status == TournamentStatusComplete {
					lastRound := tournament.Rounds[len(tournament.Rounds)-1]
					winner := lastRound.Matches[0].Winner
					sendInteractionResponse(s, i, "Tournament over!", fmt.Sprintf("The tournament is over! The winner is : %s", winner), 0x00FF00)
					return
				}

				currentRound := tournament.Rounds[tournament.CurrentRound]
				var matchesInfo strings.Builder
				matchesInfo.WriteString(fmt.Sprintf("Round %d:\n\n", tournament.CurrentRound+1))

				for _, match := range currentRound.Matches {
					if match.Player2 == "" {
						matchesInfo.WriteString(fmt.Sprintf("Match %s: %s passes automatically\n",
							match.ID, match.Player1))
					} else {
						matchesInfo.WriteString(fmt.Sprintf("Match %s: %s vs %s (Table: %s)\n",
							match.ID, match.Player1, match.Player2, match.TableID))
					}
				}

				sendInteractionResponse(s, i, "New tour begins", matchesInfo.String(), 0x00FF00)
				log.Print("Next round started successfully")
			}

		case "match":
			if len(groupCmd.Options) < 2 {
				sendInteractionResponse(s, i, "Erreur", "Match ID and winner required", 0xFF0000)
				return
			}
			matchID := groupCmd.Options[0].StringValue()
			winnerName := groupCmd.Options[1].StringValue()
			err := updateMatchResult(db, matchID, winnerName)
			if err != nil {
				sendInteractionResponse(s, i, "Erreur", "Error updating results: "+err.Error(), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Success", "Match result successfully updated!", 0x00FF00)
			log.Print("Match updated successfully")

		case "clear":
			if len(groupCmd.Options) == 0 {
				sendInteractionResponse(s, i, "Erreur", "Type of cleaning required", 0xFF0000)
				return
			}
			clearType := groupCmd.Options[0].StringValue()

			switch clearType {
			case "tournament":

				sendInteractionResponse(s, i, "Security code",
					fmt.Sprintf("To confirm the deletion of tournaments, use the command `/smashbot confirm-clear %05d`", generateSecurityCode("tournament")),
					0xFFFF00)
				return
			case "player":

				sendInteractionResponse(s, i, "Security code",
					fmt.Sprintf("To confirm the deletion of the player, use the command `/smashbot confirm-clear %05d`", generateSecurityCode("player")),
					0xFFFF00)
				return
			case "table":

				sendInteractionResponse(s, i, "Security code",
					fmt.Sprintf("To confirm the deletion of the table, use the command `/smashbot confirm-clear %05d`", generateSecurityCode("tables")),
					0xFFFF00)
				return
			case "ALL":

				sendInteractionResponse(s, i, "Security code",
					fmt.Sprintf("To confirm the deletion of the database, use the command `/smashbot confirm-clear %05d`", generateSecurityCode("database")),
					0xFFFF00)
				return
			}

			sendInteractionResponse(s, i, "Success", "Cleaning successfully completed!", 0x00FF00)
			log.Print("Database will be clear after confirmation")

		case "confirm-clear":
			if len(groupCmd.Options) < 2 {
				sendInteractionResponse(s, i, "Erreur", "Security code and type missing", 0xFF0000)
				return
			}

			securityCode := int(groupCmd.Options[0].IntValue())
			clearType := groupCmd.Options[1].StringValue()

			if err := verifySecurityCode(clearType, strconv.Itoa(securityCode)); err != nil {
				sendInteractionResponse(s, i, "Erreur", fmt.Sprintf("Incorrect security code for %s: %v", clearType, err), 0xFF0000)
				return
			}

			var (
				err        error
				successMsg string
			)
			log.Print(successMsg)
			switch clearType {
			case "tournament":
				err = clearTournament(db)
				successMsg = "Tournaments cleared successfully!"
				log.Print("Tournaments cleared successfully")
			case "player":
				err = clearPlayers(db)
				successMsg = "Players cleared successfully!"
				log.Print("Players cleared successfully")
			case "table":
				err = clearTables(db)
				successMsg = "Tables cleared successfully!"
				log.Print("Tables cleared successfully")
			case "ALL":
				err = clearDatabase(db)
				successMsg = "Database cleared successfully!"
				log.Print("Database cleared successfully")
			}

			if err != nil {
				sendInteractionResponse(s, i, "Erreur", fmt.Sprintf("Deletion error : %v", err), 0xFF0000)
				return
			}
			sendInteractionResponse(s, i, "Success", successMsg, 0x00FF00)
			log.Print("Tournaments cleared successfully")
		}
	case "help":
		helpMessage := `
**SmashBot Commands**

*Tournament Management*
- /smashbot tournament start - Start a new tournament
- /smashbot tournament next - Move to next round
- /smashbot tournament status - Display current tournament status

*Match Management*
- /smashbot match - Update match results with winner

*Player Management*
- /smashbot add player - Add new player to database
- /smashbot remove player - Remove player from database
- /smashbot list player - Display all registered players

*Table Management*
- /smashbot add tables - Add tables to venue
- /smashbot remove tables - Remove tables from venue
- /smashbot list table - Display all available tables

*Database Management*
- /smashbot clear - Clear specified data (tournament/player/table/ALL)
- /smashbot confirm-clear - Confirm clearing with security code

For more details about specific commands, use them directly to see options and requirements.`

		sendInteractionResponse(s, i, "Help - Available Commands", helpMessage, 0x00FF00)
		log.Print("Help message sent successfully")

	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("Bot token not defined in .env file")
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Bot is ready! Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		registerCommands(s)
	})

	sess.AddHandler(handleCommands)

	sess.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildIntegrations

	if err := sess.Open(); err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	log.Print("Bot is running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
