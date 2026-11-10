package database

import "fmt"

// InsertRow insère une ligne dans la table tableName et renvoie la valeur de
// la clé primaire générée (0 si la table n'a pas de clé primaire).
//
// values associe un nom de colonne à sa valeur :
//   - une colonne absente de values vaut NULL ;
//   - la clé primaire ne doit pas être fournie : le moteur la génère (1, 2, 3...).
//
// Vérifications, toutes effectuées AVANT de modifier la table (une insertion
// refusée ne laisse donc aucune trace, et ne consomme pas d'identifiant) :
//  1. la table existe ;
//  2. chaque colonne fournie existe et n'est pas la clé primaire ;
//  3. chaque colonne NOT NULL reçoit une valeur non NULL ;
//  4. chaque valeur non NULL a le type déclaré de sa colonne.
//
// Après l'ajout, chaque index de la table est mis à jour via Index.Add.
//
// Complexité : O(c + k) amorti, c = nombre de colonnes, k = nombre d'index
// (l'ajout en fin de slice est O(1) amorti ; chaque Add dépend de
// l'implémentation de l'index, O(1) pour un index de hachage).
func (db *Database) InsertRow(tableName string, values map[string]Value) (int64, error) {
	t, err := db.GetTable(tableName)
	if err != nil {
		return 0, err
	}

	// 2. Colonnes fournies : existence, et refus de la clé primaire.
	for name := range values {
		pos, ok := t.colIndex[name]
		if !ok {
			return 0, fmt.Errorf("%w : %q dans la table %q", ErrUnknownColumn, name, t.Name)
		}
		if pos == t.pkPos {
			return 0, fmt.Errorf("%w : colonne %q", ErrPrimaryKeyProvided, name)
		}
	}

	// 3 et 4. Construction de la ligne dans l'ordre du schéma.
	row := make(Row, len(t.Columns))
	for i, col := range t.Columns {
		if i == t.pkPos {
			row[i] = IntValue(t.nextID)
			continue
		}

		v, provided := values[col.Name]
		if !provided || v.Null {
			if col.NotNull {
				return 0, fmt.Errorf("%w : colonne %q de la table %q", ErrNotNullViolation, col.Name, t.Name)
			}
			row[i] = NullValue()
			continue
		}

		if v.Type != col.Type {
			return 0, fmt.Errorf("%w : colonne %q attend %s, reçu %s", ErrTypeMismatch, col.Name, col.Type, v.Type)
		}
		row[i] = normalize(v)
	}

	// Toutes les vérifications sont passées : on modifie la table.
	rowID := len(t.rows)
	t.rows = append(t.rows, row)

	var id int64
	if t.pkPos != -1 {
		id = t.nextID
		t.nextID++
	}

	// Maintien des index existants sur cette table.
	for colName, idx := range t.indexes {
		idx.Add(row[t.colIndex[colName]], rowID)
	}

	return id, nil
}

// normalize reconstruit une valeur non NULL avec son constructeur, pour ne
// garder que le champ correspondant à son type. Ainsi deux valeurs égales sont
// toujours identiques avec ==, ce dont les index de hachage ont besoin.
// Exemple : Value{Type: TypeInt, Int: 5, Str: "x"} devient IntValue(5).
func normalize(v Value) Value {
	if v.Type == TypeInt {
		return IntValue(v.Int)
	}
	return StringValue(v.Str)
}
