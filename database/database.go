// Package database implémente un SGBD relationnel simplifié, stocké en mémoire.
//
// Ce fichier contient la représentation des valeurs (cellules) et la
// structure Database qui regroupe l'ensemble des tables.
package database

import (
	"errors"
)

// Erreurs renvoyées par le moteur. Elles sont enveloppées avec %w pour
// ajouter du contexte (nom de table, de colonne...) tout en restant
// détectables avec errors.Is.
var (
	ErrInvalidName       = errors.New("nom invalide")
	ErrTableExists       = errors.New("table déjà existante")
	ErrTableNotFound     = errors.New("table introuvable")
	ErrNoColumns         = errors.New("aucune colonne définie")
	ErrDuplicateColumn   = errors.New("colonne en double")
	ErrInvalidType       = errors.New("type de colonne invalide")
	ErrInvalidPrimaryKey = errors.New("clé primaire invalide")

	ErrUnknownColumn      = errors.New("colonne inconnue")
	ErrTypeMismatch       = errors.New("type de valeur incorrect")
	ErrNotNullViolation   = errors.New("valeur obligatoire manquante")
	ErrPrimaryKeyProvided = errors.New("la clé primaire est générée par le moteur")
)

// DataType est le type déclaré d'une colonne.
type DataType int

const (
	// TypeInt représente un entier signé sur 64 bits.
	TypeInt DataType = iota
	// TypeString représente une chaîne de caractères.
	TypeString
)

// String renvoie le nom lisible du type (utile dans les messages d'erreur).
func (t DataType) String() string {
	switch t {
	case TypeInt:
		return "INT"
	case TypeString:
		return "STRING"
	default:
		return "UNKNOWN"
	}
}

// IsValid indique si le type fait partie des types supportés par le moteur.
func (t DataType) IsValid() bool {
	return t == TypeInt || t == TypeString
}

// Value représente la valeur d'une cellule : un entier, une chaîne ou NULL.
//
// On utilise une struct plutôt qu'une interface (any) pour deux raisons :
//   - le type est connu explicitement, ce qui rend la vérification à
//     l'insertion simple et sans conversion risquée ;
//   - la struct ne contient que des champs comparables, elle peut donc servir
//     directement de clé de map (indispensable pour un index de hachage).
//
// Seul le champ correspondant à Type est significatif ; l'autre reste à sa
// valeur zéro. Il faut toujours créer une Value avec IntValue, StringValue ou
// NullValue pour garantir cette règle (et donc l'égalité avec ==).
type Value struct {
	Type DataType // type de la valeur (ignoré si Null vaut true)
	Null bool     // true si la valeur est NULL
	Int  int64    // utilisé si Type == TypeInt
	Str  string   // utilisé si Type == TypeString
}

// IntValue crée une valeur entière.
func IntValue(n int64) Value {
	return Value{Type: TypeInt, Int: n}
}

// StringValue crée une valeur chaîne de caractères.
func StringValue(s string) Value {
	return Value{Type: TypeString, Str: s}
}

// NullValue crée une valeur NULL. Une valeur NULL n'a pas de type propre :
// elle est acceptée dans n'importe quelle colonne facultative.
func NullValue() Value {
	return Value{Null: true}
}

// Database regroupe les tables, indexées par leur nom.
//
// La map permet de retrouver une table en O(1) en moyenne et de détecter
// immédiatement un nom de table déjà utilisé.
type Database struct {
	tables map[string]*Table
}

// NewDatabase crée une base de données vide, prête à recevoir des tables.
func NewDatabase() *Database {
	return &Database{tables: make(map[string]*Table)}
}
