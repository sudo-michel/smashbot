Description complète du bot SmashBot:
1. Gestion des joueurs:
   * Ajouter un joueur: !smashbot add player <username>
   * Supprimer un joueur: !smashbot remove player <username>
   * Lister les joueurs: !smashbot list players
2. Gestion des tables:
   * Ajouter une table: !smashbot add table
   * Supprimer une table: !smashbot remove table
   * Lister les tables: !smashbot list tables
3. Gestion du tournoi:
   * Démarrer un tournoi: !smashbot tournament start
   * Afficher l'état du tournoi: !smashbot tournament status
   * Enregistrer le résultat d'un match: !smashbot match result <match_id> <winner_name>
4. Système de tournoi:
   * Crée des matchs initiaux en utilisant toutes les tables disponibles.
   * Les gagnants sont automatiquement mis en match contre d'autres gagnants.
   * Priorité aux joueurs n'ayant pas encore joué leur match pour accedé au meme niveau que les autre gagnant (ex on fait d'abord tout les 8e de final si une person est seul est pass en quart et une fois que tout le monde est en quart peux faire les match des quart de final et ainsi de suis jusqua la final).
   * En cas de nombre impair de joueurs, le joueur seul passe automatiquement au niveau suivant.
