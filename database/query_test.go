package database

import (
    "testing"
)

func TestIndexingAndSelectValues(t *testing.T) {
    db := NewDatabase()
    cols := []Column{
        {Name: "id", Type: TypeInt, PrimaryKey: true},
        {Name: "city", Type: TypeString},
    }
    tbl, err := db.CreateTable("users", cols)
    if err != nil {
        t.Fatalf("CreateTable: %v", err)
    }

    // Insertion avant index
    if _, err := db.InsertRow("users", map[string]Value{"city": StringValue("Paris")}); err != nil {
        t.Fatalf("Insert 1: %v", err)
    }

    // Creation de l'index optimal
    hIdx := NewHashIndex()
    if err := tbl.CreateIndex("city", hIdx); err != nil {
        t.Fatalf("CreateIndex: %v", err)
    }

    // Insertion apres index (doit appeler Add automatiquement)
    if _, err := db.InsertRow("users", map[string]Value{"city": StringValue("Lyon")}); err != nil {
        t.Fatalf("Insert 2: %v", err)
    }
    if _, err := db.InsertRow("users", map[string]Value{"city": StringValue("Paris")}); err != nil {
        t.Fatalf("Insert 3: %v", err)
    }

    // Test SelectValues avec index
    resParis, err := tbl.SelectValues("city", StringValue("Paris"))
    if err != nil {
        t.Fatalf("SelectValues Paris: %v", err)
    }
    if len(resParis) != 2 {
        t.Fatalf("Attendu 2 lignes pour Paris, recu %d", len(resParis))
    }

    // Verification que les copies ne modifient pas la base
    resParis[0][1] = StringValue("Hacked")
    resCheck, _ := tbl.SelectValues("city", StringValue("Paris"))
    if len(resCheck) != 2 {
        t.Fatalf("La donnee originale a ete alteree!")
    }

    // Test avec index sous-optimal
    llIdx := NewLinkedListIndex()
    tbl2Cols := []Column{
        {Name: "id", Type: TypeInt, PrimaryKey: true},
        {Name: "tag", Type: TypeString},
    }
    tbl2, err := db.CreateTable("tags", tbl2Cols)
    if err != nil {
        t.Fatalf("CreateTable tags: %v", err)
    }
    if err := tbl2.CreateIndex("tag", llIdx); err != nil {
        t.Fatalf("CreateIndex tags: %v", err)
    }
    _, _ = db.InsertRow("tags", map[string]Value{"tag": StringValue("go")})
    _, _ = db.InsertRow("tags", map[string]Value{"tag": StringValue("algo")})

    resTag, err := tbl2.SelectValues("tag", StringValue("go"))
    if err != nil || len(resTag) != 1 {
        t.Fatalf("Echec SelectValues avec LinkedListIndex: %v", err)
    }
}
