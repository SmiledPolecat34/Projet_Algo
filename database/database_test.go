package database

import "testing"

func TestDataTypeString(t *testing.T) {
	cases := map[DataType]string{
		TypeInt:      "INT",
		TypeString:   "STRING",
		DataType(42): "UNKNOWN",
	}
	for dt, want := range cases {
		if got := dt.String(); got != want {
			t.Errorf("DataType(%d).String() = %q, attendu %q", int(dt), got, want)
		}
	}
}

func TestDataTypeIsValid(t *testing.T) {
	if !TypeInt.IsValid() || !TypeString.IsValid() {
		t.Error("TypeInt et TypeString doivent être valides")
	}
	if DataType(42).IsValid() {
		t.Error("un type inconnu ne doit pas être valide")
	}
}

func TestValueConstructors(t *testing.T) {
	i := IntValue(2004)
	if i.Null || i.Type != TypeInt || i.Int != 2004 {
		t.Errorf("IntValue incorrect : %+v", i)
	}

	s := StringValue("Nekfeu")
	if s.Null || s.Type != TypeString || s.Str != "Nekfeu" {
		t.Errorf("StringValue incorrect : %+v", s)
	}

	n := NullValue()
	if !n.Null {
		t.Errorf("NullValue doit être NULL : %+v", n)
	}
}

// Les valeurs doivent être comparables avec == et utilisables comme clé de
// map : c'est ce qui permettra de construire des index de hachage.
func TestValueUsableAsMapKey(t *testing.T) {
	if IntValue(1) != IntValue(1) {
		t.Error("deux IntValue identiques doivent être égales")
	}
	if IntValue(1) == StringValue("1") {
		t.Error("un entier et une chaîne ne doivent pas être égaux")
	}

	m := map[Value][]int{}
	m[StringValue("Booba")] = append(m[StringValue("Booba")], 0)
	m[StringValue("Booba")] = append(m[StringValue("Booba")], 3)
	if got := len(m[StringValue("Booba")]); got != 2 {
		t.Errorf("attendu 2 positions pour la clé, obtenu %d", got)
	}
}

// fakeIndex est un index minimal, utilisé uniquement dans les tests pour
// vérifier que le contrat Index est implémentable. Ce n'est pas un vrai index :
// les implémentations réelles appartiennent à la partie « recherches et index ».
type fakeIndex struct {
	added []Value
}

func (f *fakeIndex) Add(v Value, rowID int) { f.added = append(f.added, v) }
func (f *fakeIndex) Lookup(v Value) []int   { return nil }

// Vérification à la compilation : *fakeIndex respecte l'interface Index.
var _ Index = (*fakeIndex)(nil)

func TestNewDatabase(t *testing.T) {
	db := NewDatabase()
	if db == nil || db.tables == nil {
		t.Fatal("NewDatabase doit renvoyer une base initialisée")
	}
	if len(db.tables) != 0 {
		t.Errorf("une nouvelle base doit être vide, %d tables trouvées", len(db.tables))
	}
}

func TestTableColumnIndexAndRowCount(t *testing.T) {
	// Table construite à la main : CreateTable sera implémentée à l'étape 2.
	tbl := &Table{
		Name:     "t",
		Columns:  []Column{{Name: "id", Type: TypeInt}, {Name: "name", Type: TypeString}},
		colIndex: map[string]int{"id": 0, "name": 1},
		pkPos:    0,
		rows:     []Row{{IntValue(1), StringValue("a")}},
	}

	if pos, ok := tbl.ColumnIndex("name"); !ok || pos != 1 {
		t.Errorf("ColumnIndex(name) = (%d, %v), attendu (1, true)", pos, ok)
	}
	if _, ok := tbl.ColumnIndex("absente"); ok {
		t.Error("ColumnIndex doit renvoyer false pour une colonne inconnue")
	}
	if tbl.RowCount() != 1 {
		t.Errorf("RowCount = %d, attendu 1", tbl.RowCount())
	}
}
