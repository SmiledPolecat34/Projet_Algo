package database

import (
	"errors"
	"testing"
)

// labelsSchema est un schéma d'exemple (thème du projet). Il n'est utilisé
// que dans les tests : le moteur ne connaît aucune table à l'avance.
func labelsSchema() []Column {
	return []Column{
		{Name: "id", Type: TypeInt, PrimaryKey: true},
		{Name: "name", Type: TypeString, NotNull: true},
		{Name: "annee_creation", Type: TypeInt, NotNull: true},
	}
}

func TestCreateTableOK(t *testing.T) {
	db := NewDatabase()

	tbl, err := db.CreateTable("labels", labelsSchema())
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	if tbl.Name != "labels" || len(tbl.Columns) != 3 {
		t.Errorf("table mal initialisée : %+v", tbl)
	}
	if tbl.pkPos != 0 || tbl.nextID != 1 {
		t.Errorf("pkPos = %d, nextID = %d ; attendu 0 et 1", tbl.pkPos, tbl.nextID)
	}
	if tbl.indexes == nil {
		t.Error("la map des index doit être initialisée")
	}
	if tbl.RowCount() != 0 {
		t.Errorf("une nouvelle table doit être vide")
	}
	if pos, ok := tbl.ColumnIndex("annee_creation"); !ok || pos != 2 {
		t.Errorf("ColumnIndex(annee_creation) = (%d, %v), attendu (2, true)", pos, ok)
	}

	got, err := db.GetTable("labels")
	if err != nil || got != tbl {
		t.Errorf("GetTable doit renvoyer la table créée (err = %v)", err)
	}
}

func TestCreateTableWithoutPrimaryKey(t *testing.T) {
	db := NewDatabase()
	tbl, err := db.CreateTable("logs", []Column{{Name: "message", Type: TypeString}})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if tbl.pkPos != -1 {
		t.Errorf("pkPos = %d, attendu -1 sans clé primaire", tbl.pkPos)
	}
}

func TestCreateTablePrimaryKeyIsNotNull(t *testing.T) {
	db := NewDatabase()
	// NotNull n'est pas précisé : le moteur doit l'imposer sur la clé primaire.
	tbl, err := db.CreateTable("artists", []Column{
		{Name: "id", Type: TypeInt, PrimaryKey: true},
		{Name: "name", Type: TypeString, NotNull: true},
	})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !tbl.Columns[0].NotNull {
		t.Error("la clé primaire doit être NOT NULL")
	}
}

func TestCreateTableCopiesSchema(t *testing.T) {
	db := NewDatabase()
	cols := labelsSchema()
	tbl, err := db.CreateTable("labels", cols)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	cols[1].Name = "modifie"
	if tbl.Columns[1].Name != "name" {
		t.Error("modifier la slice d'origine ne doit pas modifier la table")
	}
}

func TestCreateTableErrors(t *testing.T) {
	cases := []struct {
		name    string
		table   string
		columns []Column
		wantErr error
	}{
		{"nom de table vide", "", labelsSchema(), ErrInvalidName},
		{"nom de table avec espace", "mes labels", labelsSchema(), ErrInvalidName},
		{"nom de table commençant par un chiffre", "1labels", labelsSchema(), ErrInvalidName},
		{"aucune colonne", "vide", nil, ErrNoColumns},
		{"nom de colonne vide", "t", []Column{{Name: "", Type: TypeInt}}, ErrInvalidName},
		{"nom de colonne invalide", "t", []Column{{Name: "année", Type: TypeInt}}, ErrInvalidName},
		{
			"colonne en double", "t",
			[]Column{{Name: "name", Type: TypeString}, {Name: "name", Type: TypeInt}},
			ErrDuplicateColumn,
		},
		{"type inconnu", "t", []Column{{Name: "x", Type: DataType(99)}}, ErrInvalidType},
		{
			"clé primaire de type chaîne", "t",
			[]Column{{Name: "id", Type: TypeString, PrimaryKey: true}},
			ErrInvalidPrimaryKey,
		},
		{
			"deux clés primaires", "t",
			[]Column{
				{Name: "id", Type: TypeInt, PrimaryKey: true},
				{Name: "id2", Type: TypeInt, PrimaryKey: true},
			},
			ErrInvalidPrimaryKey,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := NewDatabase()
			_, err := db.CreateTable(tc.table, tc.columns)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erreur = %v, attendu %v", err, tc.wantErr)
			}
			// Une création refusée ne doit laisser aucune table enregistrée.
			if len(db.tables) != 0 {
				t.Errorf("aucune table ne doit être créée en cas d'erreur")
			}
		})
	}
}

func TestCreateTableDuplicateName(t *testing.T) {
	db := NewDatabase()
	if _, err := db.CreateTable("labels", labelsSchema()); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	_, err := db.CreateTable("labels", labelsSchema())
	if !errors.Is(err, ErrTableExists) {
		t.Errorf("erreur = %v, attendu ErrTableExists", err)
	}
}

func TestGetTableNotFound(t *testing.T) {
	db := NewDatabase()
	if _, err := db.GetTable("inconnue"); !errors.Is(err, ErrTableNotFound) {
		t.Errorf("erreur = %v, attendu ErrTableNotFound", err)
	}
}
