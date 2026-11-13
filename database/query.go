package database

import (
    "fmt"
)

// CreateIndex instancie ou attache un index a une colonne de la table,
// le remplit avec les lignes deja existantes, et l'enregistre dans t.indexes.
func (db *Database) CreateIndex(tableName, colName string, idx Index) error {
    t, ok := db.tables[tableName]
    if !ok {
        return fmt.Errorf("%w: table '%s'", ErrTableNotFound, tableName)
    }
    return t.CreateIndex(colName, idx)
}

func (t *Table) CreateIndex(colName string, idx Index) error {
    colPos, ok := t.colIndex[colName]
    if !ok {
        return fmt.Errorf("colonne '%s' introuvable dans la table '%s'", colName, t.Name)
    }

    if t.indexes == nil {
        t.indexes = make(map[string]Index)
    }

    // Indexation des lignes deja presentes dans la table
    for rowID, row := range t.rows {
        val := row[colPos]
        idx.Add(val, rowID)
    }

    t.indexes[colName] = idx
    return nil
}

// SelectValues recherche les lignes verifiant colName == v.
// Utilise l'index associe s'il existe (O(1) ou O(longueur liste)),
// sinon effectue un scan sequentiel (O(N)).
// Conformement aux regles, les Row renvoyees sont des copies independantes.
func (t *Table) SelectValues(colName string, v Value) ([]Row, error) {
    colPos, ok := t.colIndex[colName]
    if !ok {
        return nil, fmt.Errorf("colonne '%s' introuvable dans la table '%s'", colName, t.Name)
    }

    targetKey := toHashKey(v)

    // 1. Recherche par index si disponible
    if idx, hasIdx := t.indexes[colName]; hasIdx && idx != nil {
        rowIDs := idx.Lookup(v)
        results := make([]Row, 0, len(rowIDs))
        for _, rid := range rowIDs {
            if rid >= 0 && rid < len(t.rows) {
                results = append(results, copyRow(t.rows[rid]))
            }
        }
        return results, nil
    }

    // 2. Scan sequentiel si aucun index n'est present
    var results []Row
    for _, r := range t.rows {
        if toHashKey(r[colPos]) == targetKey {
            results = append(results, copyRow(r))
        }
    }
    return results, nil
}

func copyRow(r Row) Row {
    dup := make(Row, len(r))
    copy(dup, r)
    return dup
}
