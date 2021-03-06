// mainly by rdoc.Mutate() when applying mutations and rdoc.Traverse() when
// traversing the tree
package node

import (
	"errors"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/maps/hashmap"
	"github.com/gpestana/rdoc/clock"
	"reflect"
	_ "strconv"
)

type Node struct {
	deps []string
	// operation id that originated the node
	opId string
	// node may be a map
	hmap *hashmap.Map
	// node may be a list
	list *arraylist.List
	// node may be a register
	reg *hashmap.Map
}

func New(opId string) *Node {
	return &Node{
		deps: []string{},
		opId: opId,
		hmap: hashmap.New(),
		list: arraylist.New(),
		reg:  hashmap.New(),
	}
}

func (n *Node) AddDependency(dep string) {
	n.deps = append(n.deps, dep)
}

func (n *Node) ClearDependency(dep string) {
	n.deps = filter(n.deps, dep)
}

// returns a child node which is part of the list or map
func (n *Node) GetChild(k interface{}) (*Node, bool, error) {
	switch key := k.(type) {
	case string:
		ni, exists := n.hmap.Get(key)
		if exists {
			n := ni.(*Node)
			return n, exists, nil
		}
	case int:
		ni, exists := n.list.Get(key)
		if exists {
			n := ni.(*Node)
			return n, exists, nil
		}
	default:
		return nil, false, errors.New("Node child is stored in list or map, key must be int or string")
	}
	// child with key `k` does not exist
	return nil, false, nil
}

// returns map with all values associated to a given key. the map is indexed by
// operation ID - the operation that created the KV pair in first place
func (n *Node) GetMVRegister() map[string]interface{} {
	keys := n.reg.Keys()
	mvrMap := make(map[string]interface{})
	for _, k := range keys {
		v, _ := n.reg.Get(k)
		mvrMap[k.(string)] = v
	}
	return mvrMap
}

// adds a value to the node
func (n *Node) Add(k interface{}, v interface{}, opId string) (*Node, error) {
	var err error
	var ok bool
	var node *Node
	switch key := k.(type) {
	case string:
		// adds to map
		node, ok = v.(*Node)
		if !ok {
			node, err = newNodeWithRegisterValue(v, opId)
			if err != nil {
				return node, err
			}
		}
		n.hmap.Put(key, node)
	case int:
		// adds to list
		node, ok = v.(*Node)
		if !ok {
			node, err = newNodeWithRegisterValue(v, opId)
			if err != nil {
				return node, err
			}
		}

		// checks if element in position already exists
		// if so, calculates proper position for new elemnt in the list
		_, exists := n.list.Get(key)
		if exists {
			key = calculatePositionInsert(n.List(), node, key)
		}
		n.list.Insert(key, node)

	case nil:
		// adds to mvregister
		n.reg.Put(opId, v)
	default:
		return nil, errors.New("Key type must be of type string (map element), int (list element) or nil (register)")
	}

	return node, nil
}

// returns all direct non-leaf children (maps and lists) from node
func (n *Node) GetChildren() []*Node {
	var ich []interface{}
	ich = append(ich, n.list.Values()...)
	ich = append(ich, n.hmap.Values()...)

	ch := make([]*Node, len(ich))

	for i, c := range ich {
		ch[i] = c.(*Node)
	}
	return ch
}

func (n *Node) Deps() []string {
	return n.deps
}

func (n *Node) SetDeps(deps []string) {
	n.deps = deps
}

// testing purposes only
func (n *Node) Reg() *hashmap.Map {
	return n.reg
}

// testing purposes only
func (n *Node) Map() *hashmap.Map {
	return n.hmap
}

// testing purposes only
func (n *Node) List() *arraylist.List {
	return n.list
}

func filter(deps []string, dep string) []string {
	ndeps := []string{}
	for _, d := range deps {
		if d != dep {
			ndeps = append(ndeps, d)
		}
	}
	return ndeps
}

// creates a new node with value in register (string or int)
func newNodeWithRegisterValue(v interface{}, opId string) (*Node, error) {
	switch v.(type) {
	case string:
	case int:
	default:
		return nil, errors.New(fmt.Sprintf("register value must be int or string, got %v", reflect.TypeOf(v)))
	}
	n := New(opId)
	n.reg.Put(opId, v)
	return n, nil
}

// the real position to insert the element is relative to all operations from
// the same replica. all operations from same replica are in ascendent order
func calculatePositionInsert(list *arraylist.List, new *Node, key int) int {
	newClock, _ := clock.ConvertString(new.opId)
	var baseIndex int

	calculateBaseIndex := func() int {
		for baseIndex := 0; baseIndex < list.Size(); baseIndex++ {
			eif, _ := list.Get(baseIndex)
			elClock, _ := clock.ConvertString(eif.(*Node).opId)

			// calculate base index (index relative to the other operation elements)
			if newClock.Timestamp() > elClock.Timestamp() {
				return baseIndex
			}
		}
		return list.Size()
	}

	baseIndex = calculateBaseIndex()
	return baseIndex + key
}
