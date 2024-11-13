# SmashBot

A Discord bot designed to manage tournaments.

## Features

- Tournament Management
- Player Registration
- Table Management
- Match Result Tracking
- Double Elimination Brackets
- Automatic Table Assignment

## Prerequisites

- Go 1.20 or higher
- Discord Bot Token
- Discord Developer Account

## Installation

1. Clone the repository
```bash
git clone https://github.com/yourusername/smashbot.git
cd smashbot
```

2. Install dependencies
```bash
go mod download
```

3. Create a .env file with your Discord bot token
```bash
DISCORD_BOT_TOKEN=your_token_here
```

4. Build and run
```bash
go run .
```

## Commands

### Tournament Management
- `/smashbot tournament start` - Start a new tournament
- `/smashbot tournament next` - Move to next round
- `/smashbot tournament status` - Display current tournament status

### Match Management
- `/smashbot match [match_id] [winner]` - Update match results with winner

### Player Management
- `/smashbot add player [username]` - Add new player to database
- `/smashbot remove player [username]` - Remove player from database
- `/smashbot list player` - Display all registered players

### Table Management
- `/smashbot add tables [number]` - Add tables to venue
- `/smashbot remove tables [number]` - Remove tables from venue
- `/smashbot list table` - Display all available tables

### Database Management
- `/smashbot clear [type]` - Clear specified data (tournament/player/table/ALL)
- `/smashbot confirm-clear [code] [type]` - Confirm clearing with security code

### Help
- `/smashbot help` - Display all available commands

## Database Structure

The bot uses a JSON file (`database.json`) to store all data:
- Players: List of registered players
- Tables: Available tables for matches
- Tournaments: Tournament data including matches and rounds

## Tournament System

The tournament system follows these rules:
1. Requires minimum 2 players to start
2. Automatically creates first round matches
3. Assigns available tables to matches
4. Tracks match results
5. Generates next round matches automatically
6. Determines tournament winner


## Customization

### Change Bot Name and Command Prefix
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your application
3. Click on "Bot" in the left sidebar
4. Under "Username", click the "Edit" button
5. Enter the new name for your bot
6. Click "Save Changes"

### Change Command Prefix
To change the command prefix (default is "smashbot"), modify the following in the code:

1. Find the constant at the top of the file:
```go
const (
    BOT_COMMAND_PREFIX string = "smashbot"
)
```

2. Change it to your desired prefix:
```go
const (
    BOT_COMMAND_PREFIX string = "yourprefix"
)
```

No other changes are needed - the code uses this constant throughout. After changing:
- Your bot will respond to `/yourprefix` instead of `/smashbot`
- All commands will use the new prefix (e.g., `/yourprefix tournament start`)
- The help message will automatically show the new prefix

Note: After changing the prefix:
1. Restart the bot for changes to take effect
2. The old commands will be automatically removed
3. New commands with the new prefix will be registered

Example of commands with new prefix 'tournamentbot':
```
/tournamentbot help
/tournamentbot tournament start
/tournamentbot add player
```

