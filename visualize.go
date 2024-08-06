package light_flow

type rootTrie struct {
	tries map[string]*Trie
}

type Trie struct {
	children []*Trie
}
