// Importation des hooks depuis React
const { useState, useEffect } = React;

const Match = ({ match, round, isCurrentRound }) => {
    if (!match) return null;

    const bgColor = isCurrentRound && !match.winner ? 'bg-blue-900' : 'bg-gray-700';

    return (
        <div className={`relative ${bgColor} p-3 rounded-lg w-48 transition-colors duration-300`}>
            <div className={`${match.winner === match.player1 ? 'text-green-400' : match.winner === match.player2 ? 'text-red-400' : 'text-gray-200'} font-medium`}>
                {match.player1 || 'TBD'}
            </div>
            <div className={`${match.winner === match.player2 ? 'text-green-400' : match.winner === match.player1 ? 'text-red-400' : 'text-gray-200'} font-medium mt-1`}>
                {match.player2 || 'TBD'}
            </div>
            {match.table_id && (
                <div className="text-xs text-gray-400 mt-1">
                    Table {match.table_id}
                </div>
            )}
        </div>
    );
};

const TournamentBracket = () => {
    const [tournament, setTournament] = useState(null);
    const [error, setError] = useState(null);

    useEffect(() => {
        console.log("Fetching tournament data...");

        fetch('/api/tournament')
            .then(response => {
                console.log("Response received:", response);
                if (!response.ok) {
                    throw new Error('Failed to fetch tournament data');
                }
                return response.json();
            })
            .then(data => {
                console.log('Tournament data received:', data);
                setTournament(data);
            })
            .catch(err => {
                console.error('Error fetching data:', err);
                setError(err.message);
            });
    }, []);

    if (error) {
        return (
            <div className="min-h-screen bg-gray-900 text-gray-200 p-8">
                <div className="max-w-md mx-auto">
                    <div className="bg-red-900/50 p-4 rounded-lg">
                        <p>Error: {error}</p>
                    </div>
                </div>
            </div>
        );
    }

    if (!tournament) {
        return (
            <div className="min-h-screen bg-gray-900 text-gray-200 p-8">
                <div className="max-w-md mx-auto">
                    <p>Loading tournament data...</p>
                </div>
            </div>
        );
    }

    const players = tournament.player_ids || [];

    // Fonction pour vérifier si un joueur a perdu
    const hasPlayerLost = (playerName) => {
        return tournament.rounds.some(round =>
            round.matches.some(match =>
                match.winner && match.winner !== playerName && (match.player1 === playerName || match.player2 === playerName)
            )
        );
    };

    // Fonction pour vérifier si un joueur est toujours en jeu
    const isPlayerActive = (playerName) => {
        const currentRound = tournament.rounds[tournament.current_round];
        return currentRound.matches.some(
            match => (match.player1 === playerName || match.player2 === playerName) && !match.winner
        );
    };

    return (
        <div className="min-h-screen bg-gray-900 text-gray-200 p-8">
            <div className="flex gap-8">
                {/* Liste des joueurs */}
                <div className="w-64 flex-shrink-0">
                    <h2 className="text-xl font-bold mb-4">Players</h2>
                    <div className="bg-gray-800 rounded-lg p-4">
                        <ul className="space-y-2">
                            {players.map((player, index) => {
                                const isLost = hasPlayerLost(player);
                                const isActive = isPlayerActive(player);

                                return (
                                    <li
                                        key={index}
                                        className={`p-2 rounded flex justify-between items-center ${
                                            isLost ? 'bg-red-900/40 text-red-200' :
                                                isActive ? 'bg-blue-900 text-blue-100' :
                                                    'bg-gray-700'
                                        }`}
                                    >
                                        <span>{player}</span>
                                        <span className={`${isLost ? 'text-red-300' : 'text-gray-400'}`}>
                      #{index + 1}
                    </span>
                                    </li>
                                );
                            })}
                        </ul>
                    </div>
                </div>

                {/* Arbre du tournoi */}
                <div className="flex-grow">
                    <div className="flex justify-between items-center mb-4">
                        <h1 className="text-2xl font-bold">Tournament Bracket</h1>
                        <div className="flex gap-4 text-sm">
                            <div>Status: <span className={`font-semibold ${
                                tournament.status === 'ongoing' ? 'text-blue-400' :
                                    tournament.status === 'complete' ? 'text-green-400' :
                                        'text-gray-400'
                            }`}>{tournament.status.toUpperCase()}</span></div>
                            <div>Round: <span className="font-semibold text-blue-400">{tournament.current_round + 1}</span></div>
                        </div>
                    </div>

                    <div className="flex gap-24 items-center">
                        {tournament.rounds.map((round, roundIndex) => (
                            <div
                                key={roundIndex}
                                className={`flex flex-col gap-16 ${roundIndex === 0 ? 'mt-0' : 'mt-8'}`}
                            >
                                {round.matches.map((match, matchIndex) => (
                                    <div
                                        key={`${roundIndex}-${matchIndex}`}
                                        className="relative"
                                    >
                                        <Match
                                            match={match}
                                            round={roundIndex}
                                            isCurrentRound={roundIndex === tournament.current_round}
                                        />
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
};

// Rendre le composant disponible globalement
window.TournamentBracket = TournamentBracket;