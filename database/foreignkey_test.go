package database

import (
	"errors"
	"testing"
)

// newRapDB crée les trois tables de démonstration avec leurs clés étrangères,
// dans l'ordre imposé (une table référencée doit exister avant) :
// artists, labels, puis albums.
func newRapDB(t *testing.T) *Database {
	t.Helper()
	db := NewDatabase()

	schemas := []struct {
		name    string
		columns []Column
	}{
		{"artists", []Column{
			{Name: "id", Type: TypeInt, PrimaryKey: true},
			{Name: "name", Type: TypeString, NotNull: true},
		}},
		{"labels", labelsSchema()},
		{"albums", []Column{
			{Name: "id", Type: TypeInt, PrimaryKey: true},
			{Name: "name", Type: TypeString, NotNull: true},
			{Name: "cover", Type: TypeString},
			{Name: "artist_id", Type: TypeInt, NotNull: true, References: "artists"},
			{Name: "annee_sortie", Type: TypeInt, NotNull: true},
			{Name: "label_id", Type: TypeInt, References: "labels"},
		}},
	}
	for _, s := range schemas {
		if _, err := db.CreateTable(s.name, s.columns); err != nil {
			t.Fatalf("création de %q : %v", s.name, err)
		}
	}

	// Un artiste (id 1) et un label (id 1).
	if _, err := db.InsertRow("artists", map[string]Value{"name": StringValue("PNL")}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertRow("labels", map[string]Value{
		"name": StringValue("QLF Records"), "annee_creation": IntValue(2014),
	}); err != nil {
		t.Fatal(err)
	}
	return db
}

func album(artistID int64, labelID Value) map[string]Value {
	return map[string]Value{
		"name":         StringValue("Dans la légende"),
		"artist_id":    IntValue(artistID),
		"annee_sortie": IntValue(2016),
		"label_id":     labelID,
	}
}

func TestForeignKeyValidInsert(t *testing.T) {
	db := newRapDB(t)
	// Avec label existant.
	if _, err := db.InsertRow("albums", album(1, IntValue(1))); err != nil {
		t.Errorf("album avec artiste et label existants : %v", err)
	}
	// Label facultatif : NULL accepté sans vérification.
	if _, err := db.InsertRow("albums", album(1, NullValue())); err != nil {
		t.Errorf("album sans label : %v", err)
	}
}

func TestForeignKeyViolations(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]Value
	}{
		{"artiste inexistant", album(2, NullValue())},
		{"artiste id 0", album(0, NullValue())},
		{"artiste id négatif", album(-1, NullValue())},
		{"label inexistant", album(1, IntValue(99))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newRapDB(t)
			_, err := db.InsertRow("albums", tc.values)
			if !errors.Is(err, ErrForeignKeyViolation) {
				t.Fatalf("erreur = %v, attendu ErrForeignKeyViolation", err)
			}
			// L'insertion refusée ne laisse aucune trace.
			tbl, _ := db.GetTable("albums")
			if tbl.RowCount() != 0 {
				t.Errorf("RowCount = %d, attendu 0", tbl.RowCount())
			}
		})
	}
}

// Une référence devient valide dès que la ligne référencée est insérée.
func TestForeignKeyAfterReferencedInsert(t *testing.T) {
	db := newRapDB(t)
	if _, err := db.InsertRow("albums", album(2, NullValue())); !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("erreur = %v, attendu ErrForeignKeyViolation", err)
	}
	if _, err := db.InsertRow("artists", map[string]Value{"name": StringValue("Booba")}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertRow("albums", album(2, NullValue())); err != nil {
		t.Errorf("l'artiste 2 existe maintenant : %v", err)
	}
}

func TestForeignKeyDefinitionErrors(t *testing.T) {
	cases := []struct {
		name    string
		columns []Column
	}{
		{"table référencée inconnue", []Column{
			{Name: "x_id", Type: TypeInt, References: "inconnue"},
		}},
		{"table référencée sans clé primaire", []Column{
			{Name: "log_id", Type: TypeInt, References: "logs"},
		}},
		{"clé étrangère de type chaîne", []Column{
			{Name: "artist_id", Type: TypeString, References: "artists"},
		}},
		{"clé primaire utilisée comme clé étrangère", []Column{
			{Name: "id", Type: TypeInt, PrimaryKey: true, References: "artists"},
		}},
		{"auto-référence (table pas encore créée)", []Column{
			{Name: "id", Type: TypeInt, PrimaryKey: true},
			{Name: "parent_id", Type: TypeInt, References: "t"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := NewDatabase()
			if _, err := db.CreateTable("artists", []Column{{Name: "id", Type: TypeInt, PrimaryKey: true}}); err != nil {
				t.Fatal(err)
			}
			if _, err := db.CreateTable("logs", []Column{{Name: "msg", Type: TypeString}}); err != nil {
				t.Fatal(err)
			}

			_, err := db.CreateTable("t", tc.columns)
			if !errors.Is(err, ErrInvalidReference) {
				t.Fatalf("erreur = %v, attendu ErrInvalidReference", err)
			}
			if _, err := db.GetTable("t"); err == nil {
				t.Error("la table refusée ne doit pas être enregistrée")
			}
		})
	}
}

func TestHasID(t *testing.T) {
	db := newRapDB(t)
	artists, _ := db.GetTable("artists")
	for id, want := range map[int64]bool{-1: false, 0: false, 1: true, 2: false} {
		if got := artists.hasID(id); got != want {
			t.Errorf("hasID(%d) = %v, attendu %v", id, got, want)
		}
	}
	logs := &Table{pkPos: -1, rows: []Row{{StringValue("x")}}}
	if logs.hasID(1) {
		t.Error("une table sans clé primaire ne contient aucun id")
	}
}
