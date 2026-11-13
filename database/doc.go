// Package database implémente RapDB, un système de gestion de base de données
// relationnel simplifié, stocké en mémoire et écrit en Go sans dépendance
// externe. Il n'utilise pas SQL : on manipule la base par des appels de
// fonctions.
//
// # Principales structures
//
//   - Database : l'ensemble des tables, rangées dans une map par nom.
//   - Table : un schéma (Columns) et des lignes stockées dans une slice.
//   - Column : nom, type (TypeInt ou TypeString), NotNull, PrimaryKey et
//     References (nom de la table visée par une clé étrangère).
//   - Row : une ligne, slice de Value dans l'ordre des colonnes.
//   - Value : une cellule (entier, chaîne ou NULL), créée avec IntValue,
//     StringValue ou NullValue. Elle est comparable et peut servir de clé de map.
//   - Index : contrat (Add, Lookup) des index, mis à jour à chaque insertion.
//
// # Principales fonctions
//
//	NewDatabase() *Database
//	(*Database).CreateTable(name, columns) (*Table, error)       O(c)
//	(*Database).GetTable(name) (*Table, error)                   O(1)
//	(*Database).InsertRow(table, values) (id int64, err error)   O(c + k)
//	(*Table).ColumnIndex(name) (int, bool)                       O(1)
//	(*Table).RowCount() int                                      O(1)
//
// (c = nombre de colonnes, k = nombre d'index de la table.)
//
// La clé primaire est un entier généré par le moteur (1, 2, 3...) : elle ne
// doit jamais être fournie à InsertRow. Une colonne absente de values vaut NULL.
//
// # Exemple : artistes, labels et albums
//
// Une table référencée par une clé étrangère doit exister avant la table qui
// la référence. L'ordre de création est donc : artists, labels, puis albums.
//
//	db := database.NewDatabase()
//
//	db.CreateTable("artists", []database.Column{
//		{Name: "id", Type: database.TypeInt, PrimaryKey: true},
//		{Name: "name", Type: database.TypeString, NotNull: true},
//	})
//	db.CreateTable("labels", []database.Column{
//		{Name: "id", Type: database.TypeInt, PrimaryKey: true},
//		{Name: "name", Type: database.TypeString, NotNull: true},
//		{Name: "annee_creation", Type: database.TypeInt, NotNull: true},
//	})
//	db.CreateTable("albums", []database.Column{
//		{Name: "id", Type: database.TypeInt, PrimaryKey: true},
//		{Name: "name", Type: database.TypeString, NotNull: true},
//		{Name: "cover", Type: database.TypeString},
//		{Name: "artist_id", Type: database.TypeInt, NotNull: true, References: "artists"},
//		{Name: "annee_sortie", Type: database.TypeInt, NotNull: true},
//		{Name: "label_id", Type: database.TypeInt, References: "labels"},
//	})
//
//	artistID, _ := db.InsertRow("artists", map[string]database.Value{
//		"name": database.StringValue("PNL"),
//	})
//	labelID, _ := db.InsertRow("labels", map[string]database.Value{
//		"name":           database.StringValue("QLF Records"),
//		"annee_creation": database.IntValue(2014),
//	})
//	_, err := db.InsertRow("albums", map[string]database.Value{
//		"name":         database.StringValue("Dans la légende"),
//		"artist_id":    database.IntValue(artistID),
//		"annee_sortie": database.IntValue(2016),
//		"label_id":     database.IntValue(labelID), // facultatif : peut être omis
//	})
//	if err != nil {
//		// errors.Is(err, database.ErrForeignKeyViolation), etc.
//	}
//
// Pour rester lisible, l'exemple ignore les erreurs de CreateTable : dans un
// vrai programme, il faut les vérifier.
//
// # Erreurs
//
// Les erreurs sont enveloppées avec un message de contexte et se testent avec
// errors.Is :
//
//   - CreateTable : ErrInvalidName, ErrTableExists, ErrNoColumns,
//     ErrDuplicateColumn, ErrInvalidType, ErrInvalidPrimaryKey,
//     ErrInvalidReference.
//   - InsertRow : ErrTableNotFound, ErrUnknownColumn, ErrPrimaryKeyProvided,
//     ErrNotNullViolation, ErrTypeMismatch, ErrForeignKeyViolation.
//   - GetTable : ErrTableNotFound.
//
// Une opération refusée ne modifie jamais la base (aucune ligne ajoutée,
// aucun identifiant consommé).
//
// # Notes pour l'intégration (recherches et index)
//
//   - Le rowID d'une ligne est sa position dans la table ; il est stable car
//     le moteur ne supprime jamais de ligne.
//   - CreateIndex doit remplir l'index avec les lignes existantes puis
//     l'enregistrer dans la table : InsertRow appellera ensuite Add
//     automatiquement à chaque insertion.
//   - Les Row renvoyées par SelectValues doivent être des copies, pour
//     empêcher toute modification directe des données internes.
//   - Les cellules NULL sont transmises à Add (avec NullValue()) : chaque index
//     doit les traiter de manière cohérente (les ignorer ou les regrouper).
//   - Les valeurs recherchées doivent être normalisées (comme le fait
//     InsertRow) avant Lookup, sans quoi une valeur égale peut ne pas être
//     trouvée.
//
// Limites : stockage uniquement en mémoire, pas de suppression ni de mise à
// jour, pas de transactions ni d'accès concurrent.
package database
