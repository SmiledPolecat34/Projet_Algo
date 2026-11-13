package database

import (
    "fmt"
    "testing"
)

// setupBenchmarkTable prepare une table avec N lignes
func setupBenchmarkTable(b *testing.B, n int) (*Database, *Table) {
    b.Helper()
    db := NewDatabase()
    cols := []Column{
        {Name: "id", Type: TypeInt, PrimaryKey: true},
        {Name: "department", Type: TypeString},
        {Name: "salary", Type: TypeInt},
    }
    tbl, err := db.CreateTable("employees", cols)
    if err != nil {
        b.Fatalf("CreateTable: %v", err)
    }

    // N lignes reparties sur 10 departements (10% par departement)
    for i := 0; i < n; i++ {
        dept := fmt.Sprintf("Dept_%02d", i%10)
        _, err := db.InsertRow("employees", map[string]Value{
            "department": StringValue(dept),
            "salary":     IntValue(int64(30000 + (i % 50000))),
        })
        if err != nil {
            b.Fatalf("InsertRow %d: %v", i, err)
        }
    }
    return db, tbl
}

// 1. Recherche directe avec HashIndex (Optimal - O(1))
func BenchmarkSelectWithHashIndex(b *testing.B) {
    _, tbl := setupBenchmarkTable(b, 10000)

    hIdx := NewHashIndex()
    if err := tbl.CreateIndex("department", hIdx); err != nil {
        b.Fatalf("CreateIndex: %v", err)
    }

    target := StringValue("Dept_05")

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        res, err := tbl.SelectValues("department", target)
        if err != nil || len(res) != 1000 {
            b.Fatalf("SelectValues error or bad count: %v, got %d", err, len(res))
        }
    }
}

// 2. Recherche avec LinkedListIndex (Sous-optimal - O(N))
func BenchmarkSelectWithLinkedListIndex(b *testing.B) {
    _, tbl := setupBenchmarkTable(b, 10000)

    llIdx := NewLinkedListIndex()
    if err := tbl.CreateIndex("department", llIdx); err != nil {
        b.Fatalf("CreateIndex: %v", err)
    }

    target := StringValue("Dept_05")

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        res, err := tbl.SelectValues("department", target)
        if err != nil || len(res) != 1000 {
            b.Fatalf("SelectValues error or bad count: %v, got %d", err, len(res))
        }
    }
}

// 3. Recherche sans aucun index (Full Table Scan - O(N))
func BenchmarkSelectFullScan(b *testing.B) {
    _, tbl := setupBenchmarkTable(b, 10000)

    target := StringValue("Dept_05")

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        res, err := tbl.SelectValues("department", target)
        if err != nil || len(res) != 1000 {
            b.Fatalf("SelectValues error or bad count: %v, got %d", err, len(res))
        }
    }
}
