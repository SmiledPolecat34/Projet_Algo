package database

// Column décrit une colonne du schéma d'une table.
type Column struct {
	Name       string   // nom unique dans la table
	Type       DataType // type des valeurs acceptées
	NotNull    bool     // true si la valeur est obligatoire
	PrimaryKey bool     // clé primaire entière, auto-incrémentée par le moteur
	References string   // table référencée par une clé étrangère ("" = aucune)
}

// Row est une ligne de la table.
//
// Elle est positionnelle : Row[i] contient la valeur de Columns[i]. Ce choix
// est plus compact qu'une map[string]Value : le nom des colonnes n'est stocké
// qu'une seule fois, dans le schéma.
type Row []Value

// Index est le contrat minimal que doit respecter un index pour être tenu à
// jour automatiquement lors des insertions.
//
// Les implémentations concrètes (index de hachage, index trié, ...) et les
// fonctions de recherche appartiennent à la partie « recherches et index ».
// Le moteur de stockage se contente d'appeler Add après chaque insertion.
type Index interface {
	// Add enregistre que la ligne située à la position rowID contient la
	// valeur v dans la colonne indexée.
	Add(v Value, rowID int)
}

// Table contient le schéma (colonnes) et les données (lignes) d'une table.
//
// Identifiant interne d'une ligne (rowID) : sa position dans rows. Comme le
// moteur ne gère pas la suppression, cette position est stable et peut être
// stockée dans les index.
type Table struct {
	Name    string
	Columns []Column

	colIndex map[string]int   // nom de colonne -> position dans Columns (O(1))
	pkPos    int              // position de la clé primaire, -1 si aucune
	rows     []Row            // lignes stockées, dans l'ordre d'insertion
	nextID   int64            // prochaine valeur de la clé primaire
	indexes  map[string]Index // nom de colonne -> index associé
}

// ColumnIndex renvoie la position de la colonne name dans le schéma.
// Le booléen vaut false si la colonne n'existe pas.
func (t *Table) ColumnIndex(name string) (int, bool) {
	pos, ok := t.colIndex[name]
	return pos, ok
}

// RowCount renvoie le nombre de lignes stockées dans la table.
func (t *Table) RowCount() int {
	return len(t.rows)
}
