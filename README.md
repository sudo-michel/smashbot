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



Initialisation du tournoi :
a. Déterminer la plus grande puissance de 2 inférieure ou égale au nombre de joueurs.
b. Créer des matchs réguliers pour cette puissance de 2.
c. Gérer les joueurs restants en mini-tournois.
Création des matchs :
a. Matchs réguliers :

Créer des paires de joueurs pour des matchs directs.
Le nombre de ces matchs sera la moitié de la puissance de 2 trouvée.

b. Mini-tournois :

Pour chaque groupe de 3 joueurs restants, créer un mini-tournoi.
Si le nombre de joueurs restants n'est pas divisible par 3, le(s) dernier(s) joueur(s) forme(nt) un match direct ou obtient un "bye".


Structure d'un mini-tournoi :

Premier match : Joueur A vs Joueur B
Deuxième match : Gagnant (A vs B) contre Joueur C


Déroulement du tournoi :
a. Jouer tous les matchs réguliers du tour actuel.
b. Pour chaque mini-tournoi :

Jouer le premier match.
Une fois le résultat connu, créer et jouer le deuxième match.
c. Si présent, jouer le match direct des joueurs restants.


Progression au tour suivant :
a. Collecter tous les gagnants des matchs réguliers.
b. Ajouter les gagnants finaux de chaque mini-tournoi.
c. Inclure le gagnant du match direct des joueurs restants, s'il y en a un.
d. Si un joueur a reçu un "bye", l'inclure directement.
