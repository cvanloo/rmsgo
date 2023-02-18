package filetree

import (
	"testing"
)

func TestAdd(t *testing.T) {
	nodes = make(map[string]NodeInfo)
	const name = "/someuser/pictures/kitten.png"
	doc, err := NewDocument(name, "image/png")
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}
	Add(doc)

	node, found := Get(name)
	if !found {
		t.Fatal("expected node to exist")
	}
	_ = node
}

func TestRemove(t *testing.T) {
	nodes = make(map[string]NodeInfo)
	const name = "/someuser/pictures/kitten.png"
	doc, err := NewDocument(name, "image/png")
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}
	Add(doc)

	node, found := Get(name)
	if !found {
		t.Fatal("expected node to exist")
	}

	Remove(name)
	node, found = Get(name)
	if found {
		t.Fatal("expected node to have been removed")
	}

	_ = node
}
