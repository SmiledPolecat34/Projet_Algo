package database

// HashKey permet d'utiliser une Value comme clef de map Go.
type HashKey struct {
    Null bool
    Type DataType
    Int  int64
    Str  string
}

func toHashKey(v Value) HashKey {
    if v.Null {
        return HashKey{Null: true}
    }
    return HashKey{
        Null: false,
        Type: v.Type,
        Int:  v.Int,
        Str:  v.Str,
    }
}

// =====================================================================
// 1. Index optimal : HashIndex (O(1))
// =====================================================================

type HashIndex struct {
    entries map[HashKey][]int
}

func NewHashIndex() *HashIndex {
    return &HashIndex{
        entries: make(map[HashKey][]int),
    }
}

func (idx *HashIndex) Add(v Value, rowID int) {
    k := toHashKey(v)
    idx.entries[k] = append(idx.entries[k], rowID)
}

func (idx *HashIndex) Lookup(v Value) []int {
    k := toHashKey(v)
    res := idx.entries[k]
    if len(res) == 0 {
        return nil
    }
    out := make([]int, len(res))
    copy(out, res)
    return out
}

// =====================================================================
// 2. Index sous-optimal : LinkedListIndex (O(N))
// =====================================================================

type listNode struct {
    val   Value
    rowID int
    next  *listNode
}

type LinkedListIndex struct {
    head *listNode
}

func NewLinkedListIndex() *LinkedListIndex {
    return &LinkedListIndex{head: nil}
}

func (idx *LinkedListIndex) Add(v Value, rowID int) {
    idx.head = &listNode{
        val:   v,
        rowID: rowID,
        next:  idx.head,
    }
}

func (idx *LinkedListIndex) Lookup(v Value) []int {
    var result []int
    curr := idx.head
    kTarget := toHashKey(v)
    for curr != nil {
        if toHashKey(curr.val) == kTarget {
            result = append(result, curr.rowID)
        }
        curr = curr.next
    }
    return result
}
