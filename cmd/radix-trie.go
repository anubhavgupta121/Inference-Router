package main

import (
	"slices"
	"sort"
)

type edge struct {
	label byte
	node  *node
}
type edges []edge

type node struct {
	server *Server
	prefix string
	edges  edges
}

type Tree struct {
	root *node
}

func Generate_Rad_Tree() *Tree {
	return &Tree{}
}

func (n *node) getNode(label byte) *node {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		return n.edges[idx].node
	}
	return nil
}

func (n *node) addEdge(e edge) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= e.label
	})

	n.edges = append(n.edges, edge{})
	copy(n.edges[idx+1:], n.edges[idx:])
	n.edges[idx] = e
}

func (n *node) DelEdge(e edge) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= e.label
	})

	if idx < num && n.edges[idx].label == e.label {
		n.edges = slices.Delete(n.edges, idx, idx+1)
	}

}

func longestP(s1 string, s2 string) int {
	i := 0
	for i < min(len(s1), len(s2)) {
		if s1[i] != s2[i] {
			return i
		}
		i++
	}
	return i
}

func (t *Tree) Insert(s string) (*node, *Server) {
	if t.root == nil {
		t.root = &node{prefix: s}
		return t.root, nil
	}
	var parent *node = nil
	n := t.root
	search := s
	var server_ *Server = nil
	var retnode *node = nil
	for true {

		if n.server != nil {
			server_ = n.server
		}
		comm_p := longestP(search, n.prefix)

		if comm_p == len(n.prefix) {
			search = search[comm_p:]
			if search != "" {
				new_node := n.getNode(search[0])
				if new_node != nil {
					parent = n
					n = new_node
					continue
				} else {
					new_edge := edge{label: search[0], node: &node{server: nil, prefix: search}}
					n.addEdge(new_edge)
					retnode = new_edge.node
					break
				}
			} else {
				retnode = n
				return retnode, server_
			}
		} else {
			split_curr := n.prefix[comm_p:]
			split_search := search[comm_p:]

			common := n.prefix[:comm_p]

			comm_node := &node{server: nil, prefix: common}
			n.prefix = split_curr
			comm_node.addEdge(edge{label: n.prefix[0], node: n})
			retnode = comm_node

			if split_search != "" {
				search_node := &node{server: nil, prefix: split_search}
				comm_node.addEdge(edge{label: search_node.prefix[0], node: search_node})
				retnode = search_node
			}
			if parent != nil {
				parent.DelEdge(edge{label: common[0], node: n})
				parent.addEdge(edge{label: comm_node.prefix[0], node: comm_node})
			} else {
				t.root = comm_node
			}

			break

		}
	}
	return retnode, server_
}
