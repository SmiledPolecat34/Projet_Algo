package database

import "fmt"

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

// Index est le contrat commun à tous les index d'une colonne.
//
// Les implémentations concrètes (index de hachage, index trié, ...) et les
// fonctions de recherche appartiennent à la partie « recherches et index ».
// Le moteur de stockage se contente d'appeler Add après chaque insertion ;
// Lookup est utilisé par les recherches (SelectValues).
type Index interface {
	// Add enregistre que la ligne située à la position rowID contient la
	// valeur v dans la colonne indexée.
	Add(v Value, rowID int)

	// Lookup renvoie les positions (rowID) des lignes dont la colonne
	// indexée vaut exactement v. Renvoie une slice vide (ou nil) si aucune
	// ligne ne correspond.
	Lookup(v Value) []int
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

// CreateTable crée une table vide à partir de son nom et de son schéma.
//
// Règles vérifiées :
//   - le nom de la table est un identifiant valide et n'est pas déjà utilisé ;
//   - il y a au moins une colonne ;
//   - chaque colonne a un nom valide, unique dans la table, et un type connu ;
//   - au plus une colonne est clé primaire, et elle est de type TypeInt ;
//   - chaque clé étrangère (References) est correctement définie
//     (voir checkReference).
//
// Une clé primaire est toujours considérée comme NOT NULL : sa valeur sera
// générée par le moteur (1, 2, 3, ...) lors des insertions.
//
// Complexité : O(c), où c est le nombre de colonnes (une passe sur le schéma,
// chaque vérification de doublon étant un accès O(1) à une map).
func (db *Database) CreateTable(name string, columns []Column) (*Table, error) {
	if !isValidName(name) {
		return nil, fmt.Errorf("%w : table %q", ErrInvalidName, name)
	}
	if _, exists := db.tables[name]; exists {
		return nil, fmt.Errorf("%w : %q", ErrTableExists, name)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("%w : table %q", ErrNoColumns, name)
	}

	// Copie du schéma : l'appelant peut modifier sa slice ensuite sans
	// altérer la table.
	cols := make([]Column, len(columns))
	copy(cols, columns)

	colIndex := make(map[string]int, len(cols))
	pkPos := -1

	for i := range cols {
		col := &cols[i]

		if !isValidName(col.Name) {
			return nil, fmt.Errorf("%w : colonne %q de la table %q", ErrInvalidName, col.Name, name)
		}
		if _, dup := colIndex[col.Name]; dup {
			return nil, fmt.Errorf("%w : %q dans la table %q", ErrDuplicateColumn, col.Name, name)
		}
		if !col.Type.IsValid() {
			return nil, fmt.Errorf("%w : colonne %q (%d)", ErrInvalidType, col.Name, int(col.Type))
		}

		if col.PrimaryKey {
			if pkPos != -1 {
				return nil, fmt.Errorf("%w : plusieurs clés primaires dans la table %q", ErrInvalidPrimaryKey, name)
			}
			if col.Type != TypeInt {
				return nil, fmt.Errorf("%w : la colonne %q doit être de type INT", ErrInvalidPrimaryKey, col.Name)
			}
			col.NotNull = true
			pkPos = i
		}

		if col.References != "" {
			if err := db.checkReference(name, col); err != nil {
				return nil, err
			}
		}

		colIndex[col.Name] = i
	}

	t := &Table{
		Name:     name,
		Columns:  cols,
		colIndex: colIndex,
		pkPos:    pkPos,
		nextID:   1,
		indexes:  make(map[string]Index),
	}
	db.tables[name] = t
	return t, nil
}

// checkReference vérifie la définition d'une clé étrangère (colonne dont le
// champ References est renseigné) lors de la création de la table tableName :
//   - la table référencée existe déjà (elle doit donc être créée avant) ;
//   - elle possède une clé primaire, vers laquelle pointe la référence ;
//   - la colonne est de type INT, comme toute clé primaire du moteur ;
//   - la colonne n'est pas elle-même la clé primaire (générée par le moteur).
func (db *Database) checkReference(tableName string, col *Column) error {
	ref, ok := db.tables[col.References]
	if !ok {
		return fmt.Errorf("%w : %s.%s référence la table inconnue %q",
			ErrInvalidReference, tableName, col.Name, col.References)
	}
	if ref.pkPos == -1 {
		return fmt.Errorf("%w : la table %q n'a pas de clé primaire", ErrInvalidReference, ref.Name)
	}
	if col.Type != TypeInt {
		return fmt.Errorf("%w : %s.%s doit être de type INT", ErrInvalidReference, tableName, col.Name)
	}
	if col.PrimaryKey {
		return fmt.Errorf("%w : la clé primaire %s.%s ne peut pas être une clé étrangère",
			ErrInvalidReference, tableName, col.Name)
	}
	return nil
}

// hasID indique si la table contient une ligne dont la clé primaire vaut id.
//
// Complexité O(1), sans parcours ni index : les identifiants sont générés
// séquentiellement à partir de 1 et aucune ligne n'est jamais supprimée, donc
// les identifiants existants sont exactement 1, 2, ..., RowCount() (la ligne
// d'id k étant à la position k-1). Si la suppression était ajoutée un jour,
// cette fonction devrait utiliser un index sur la clé primaire.
func (t *Table) hasID(id int64) bool {
	return t.pkPos != -1 && id >= 1 && id <= int64(len(t.rows))
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
