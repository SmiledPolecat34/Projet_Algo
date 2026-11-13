package database

import (
	"errors"
	"testing"
)

// newAlbumsDB crée une base contenant une table d'exemple « albums ».
// Les clés étrangères ne sont pas encore vérifiées (étape suivante).
func newAlbumsDB(t *testing.T) *Database {
	t.Helper()
	db := NewDatabase()
	_, err := db.CreateTable("albums", []Column{
		{Name: "id", Type: TypeInt, PrimaryKey: true},
		{Name: "name", Type: TypeString, NotNull: true},
		{Name: "cover", Type: TypeString},
		{Name: "artist_id", Type: TypeInt, NotNull: true},
		{Name: "annee_sortie", Type: TypeInt, NotNull: true},
		{Name: "label_id", Type: TypeInt},
	})
	if err != nil {
		t.Fatalf("création de la table : %v", err)
	}
	return db
}

// validAlbum renvoie une ligne valide (sans les colonnes facultatives).
func validAlbum() map[string]Value {
	return map[string]Value{
		"name":         StringValue("Ipséité"),
		"artist_id":    IntValue(1),
		"annee_sortie": IntValue(2017),
	}
}

func TestInsertRowGeneratesIDs(t *testing.T) {
	db := newAlbumsDB(t)
	for want := int64(1); want <= 3; want++ {
		id, err := db.InsertRow("albums", validAlbum())
		if err != nil {
			t.Fatalf("insertion %d : %v", want, err)
		}
		if id != want {
			t.Errorf("id = %d, attendu %d", id, want)
		}
	}

	tbl, _ := db.GetTable("albums")
	if tbl.RowCount() != 3 {
		t.Fatalf("RowCount = %d, attendu 3", tbl.RowCount())
	}
	// La ligne d'id k est stockée à la position k-1.
	for pos, row := range tbl.rows {
		if row[0] != IntValue(int64(pos+1)) {
			t.Errorf("ligne %d : id = %+v, attendu %d", pos, row[0], pos+1)
		}
	}
}

func TestInsertRowStoresValuesInSchemaOrder(t *testing.T) {
	db := newAlbumsDB(t)
	values := validAlbum()
	values["cover"] = StringValue("ipseite.jpg")
	values["label_id"] = IntValue(4)

	if _, err := db.InsertRow("albums", values); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	tbl, _ := db.GetTable("albums")
	want := Row{
		IntValue(1), StringValue("Ipséité"), StringValue("ipseite.jpg"),
		IntValue(1), IntValue(2017), IntValue(4),
	}
	got := tbl.rows[0]
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("colonne %s : %+v, attendu %+v", tbl.Columns[i].Name, got[i], want[i])
		}
	}
}

func TestInsertRowOptionalColumnsAreNull(t *testing.T) {
	db := newAlbumsDB(t)
	values := validAlbum()
	values["label_id"] = NullValue() // NULL explicite
	// "cover" est omise : NULL implicite

	if _, err := db.InsertRow("albums", values); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	tbl, _ := db.GetTable("albums")
	row := tbl.rows[0]
	coverPos, _ := tbl.ColumnIndex("cover")
	labelPos, _ := tbl.ColumnIndex("label_id")
	if !row[coverPos].Null || !row[labelPos].Null {
		t.Errorf("cover et label_id doivent être NULL : %+v", row)
	}
}

func TestInsertRowNormalizesValues(t *testing.T) {
	db := newAlbumsDB(t)
	values := validAlbum()
	// Valeur construite à la main avec un champ parasite.
	values["annee_sortie"] = Value{Type: TypeInt, Int: 2017, Str: "parasite"}

	if _, err := db.InsertRow("albums", values); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	tbl, _ := db.GetTable("albums")
	pos, _ := tbl.ColumnIndex("annee_sortie")
	if tbl.rows[0][pos] != IntValue(2017) {
		t.Errorf("valeur non normalisée : %+v", tbl.rows[0][pos])
	}
}

func TestInsertRowTableWithoutPrimaryKey(t *testing.T) {
	db := NewDatabase()
	if _, err := db.CreateTable("logs", []Column{{Name: "msg", Type: TypeString}}); err != nil {
		t.Fatal(err)
	}
	id, err := db.InsertRow("logs", map[string]Value{"msg": StringValue("ok")})
	if err != nil || id != 0 {
		t.Errorf("InsertRow = (%d, %v), attendu (0, nil)", id, err)
	}
	// Une map nil est acceptée : toutes les colonnes valent alors NULL.
	if _, err := db.InsertRow("logs", nil); err != nil {
		t.Errorf("insertion avec map nil : %v", err)
	}
}

func TestInsertRowErrors(t *testing.T) {
	cases := []struct {
		name    string
		table   string
		modify  func(map[string]Value)
		wantErr error
	}{
		{"table inconnue", "inconnue", func(map[string]Value) {}, ErrTableNotFound},
		{"colonne inconnue", "albums", func(v map[string]Value) { v["genre"] = StringValue("rap") }, ErrUnknownColumn},
		{"id fourni", "albums", func(v map[string]Value) { v["id"] = IntValue(42) }, ErrPrimaryKeyProvided},
		{"NOT NULL omise", "albums", func(v map[string]Value) { delete(v, "name") }, ErrNotNullViolation},
		{"NOT NULL à NULL", "albums", func(v map[string]Value) { v["artist_id"] = NullValue() }, ErrNotNullViolation},
		{"chaîne dans un INT", "albums", func(v map[string]Value) { v["annee_sortie"] = StringValue("2017") }, ErrTypeMismatch},
		{"entier dans un STRING", "albums", func(v map[string]Value) { v["cover"] = IntValue(3) }, ErrTypeMismatch},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newAlbumsDB(t)
			values := validAlbum()
			tc.modify(values)

			_, err := db.InsertRow(tc.table, values)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erreur = %v, attendu %v", err, tc.wantErr)
			}
		})
	}
}

// Une insertion refusée ne doit ni ajouter de ligne ni consommer d'identifiant.
func TestInsertRowFailureLeavesTableUnchanged(t *testing.T) {
	db := newAlbumsDB(t)
	bad := validAlbum()
	bad["annee_sortie"] = StringValue("pas un entier")
	if _, err := db.InsertRow("albums", bad); err == nil {
		t.Fatal("l'insertion invalide aurait dû échouer")
	}

	tbl, _ := db.GetTable("albums")
	if tbl.RowCount() != 0 {
		t.Errorf("RowCount = %d, attendu 0", tbl.RowCount())
	}
	id, err := db.InsertRow("albums", validAlbum())
	if err != nil || id != 1 {
		t.Errorf("InsertRow = (%d, %v), attendu (1, nil)", id, err)
	}
}

// Les index enregistrés sur la table reçoivent chaque nouvelle ligne.
func TestInsertRowUpdatesIndexes(t *testing.T) {
	db := newAlbumsDB(t)
	tbl, _ := db.GetTable("albums")

	// Index factice posé à la main : CreateIndex appartient à la partie 2.
	idx := &fakeIndex{}
	tbl.indexes["annee_sortie"] = idx

	years := []int64{2017, 2020}
	for _, y := range years {
		v := validAlbum()
		v["annee_sortie"] = IntValue(y)
		if _, err := db.InsertRow("albums", v); err != nil {
			t.Fatal(err)
		}
	}

	if len(idx.added) != 2 {
		t.Fatalf("Add appelé %d fois, attendu 2", len(idx.added))
	}
	for i, y := range years {
		if idx.added[i] != IntValue(y) || idx.rowIDs[i] != i {
			t.Errorf("appel %d : Add(%+v, %d), attendu Add(%d, %d)", i, idx.added[i], idx.rowIDs[i], y, i)
		}
	}
}
