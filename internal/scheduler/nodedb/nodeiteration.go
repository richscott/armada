package nodedb

import (
	"bytes"
	"container/heap"

	"github.com/hashicorp/go-memdb"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	log "github.com/armadaproject/armada/internal/common/logging"
	"github.com/armadaproject/armada/internal/scheduler/internaltypes"
)

type NodeIterator interface {
	NextNode() *internaltypes.Node
}

// NodesIterator is an iterator over all nodes in the db.
type NodesIterator struct {
	it memdb.ResultIterator
}

func NewNodesIterator(txn *memdb.Txn) (*NodesIterator, error) {
	it, err := txn.LowerBound("nodes", "id", "")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &NodesIterator{
		it: it,
	}, nil
}

func (it *NodesIterator) WatchCh() <-chan struct{} {
	panic("not implemented")
}

func (it *NodesIterator) NextNode() *internaltypes.Node {
	obj := it.it.Next()
	if obj == nil {
		return nil
	}
	return obj.(*internaltypes.Node)
}

func (it *NodesIterator) Next() interface{} {
	return it.NextNode()
}

// NodeIndex is an index for internaltypes.Node that returns node.NodeDbKeys[KeyIndex].
type NodeIndex struct {
	KeyIndex int
}

// FromArgs computes the index key from a set of arguments.
// Takes a single argument resourceAmount of type []byte.
func (index *NodeIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("must provide exactly one argument")
	}
	return args[0].([]byte), nil
}

// FromObject extracts the index key from a *Node.
func (index *NodeIndex) FromObject(raw interface{}) (bool, []byte, error) {
	node := raw.(*internaltypes.Node)
	return true, node.Keys[index.KeyIndex], nil
}

// NodeTypesIterator is an iterator over all nodes of the given nodeTypes
// with at least some specified amount of resources allocatable at a given priority.
// For example, all nodes of nodeType "foo" and "bar" with at least 2 cores and 1Gi memory allocatable at priority 2.
// Nodes are returned in sorted order, from least to most of the specified resource available.
type NodeTypesIterator struct {
	pq *nodeTypesIteratorPQ
}

func NewNodeTypesIterator(
	txn *memdb.Txn,
	nodeTypeIds []uint64,
	indexName string,
	priority int32,
	keyIndex int,
	indexedResources []string,
	indexedResourceRequests []int64,
	indexedResourceResolution []int64,
) (*NodeTypesIterator, error) {
	pq := &nodeTypesIteratorPQ{
		priority:         priority,
		indexedResources: indexedResources,
		items:            make([]*nodeTypesIteratorPQItem, 0, len(nodeTypeIds)),
	}
	for _, nodeTypeId := range nodeTypeIds {
		it, err := NewNodeTypeIterator(
			txn,
			nodeTypeId,
			indexName,
			priority,
			keyIndex,
			indexedResources,
			indexedResourceRequests,
			indexedResourceResolution,
		)
		if err != nil {
			return nil, err
		}
		node, err := it.NextNode()
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		heap.Push(pq, &nodeTypesIteratorPQItem{
			node: node,
			it:   it,
		})
	}
	return &NodeTypesIterator{pq: pq}, nil
}

func (it *NodeTypesIterator) WatchCh() <-chan struct{} {
	panic("not implemented")
}

func (it *NodeTypesIterator) Next() interface{} {
	v, err := it.NextNode()
	if err != nil {
		panic(err)
	}
	return v
}

func (it *NodeTypesIterator) NextNode() (*internaltypes.Node, error) {
	if it.pq.Len() == 0 {
		return nil, nil
	}
	pqItem := heap.Pop(it.pq).(*nodeTypesIteratorPQItem)
	node := pqItem.node
	nextNode, err := pqItem.it.NextNode()
	if err != nil {
		return nil, err
	}
	if nextNode != nil {
		pqItem.node = nextNode
		heap.Push(it.pq, pqItem)
	}
	return node, nil
}

type nodeTypesIteratorPQ struct {
	priority         int32
	indexedResources []string
	items            []*nodeTypesIteratorPQItem
}

type nodeTypesIteratorPQItem struct {
	node *internaltypes.Node
	it   *NodeTypeIterator
	// The index of the item in the heap. Maintained by the heap.Interface methods.
	index int
}

func (pq *nodeTypesIteratorPQ) Len() int { return len(pq.items) }

func (pq *nodeTypesIteratorPQ) Less(i, j int) bool {
	return pq.less(pq.items[i].node, pq.items[j].node)
}

func (it *nodeTypesIteratorPQ) less(a, b *internaltypes.Node) bool {
	allocatableByPriorityA := a.AllocatableByPriority[it.priority]
	allocatableByPriorityB := b.AllocatableByPriority[it.priority]
	for _, t := range it.indexedResources {
		qa := allocatableByPriorityA.GetByNameZeroIfMissing(t)
		qb := allocatableByPriorityB.GetByNameZeroIfMissing(t)

		if qa < qb {
			return true
		} else if qa > qb {
			return false
		}
	}
	// Tie-break by id.
	return a.GetId() < b.GetId()
}

func (pq *nodeTypesIteratorPQ) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *nodeTypesIteratorPQ) Push(x any) {
	n := len(pq.items)
	item := x.(*nodeTypesIteratorPQItem)
	item.index = n
	pq.items = append(pq.items, item)
}

func (pq *nodeTypesIteratorPQ) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	pq.items = old[0 : n-1]
	return item
}

// NodeTypeIterator is an iterator over all nodes of a given nodeType
// with at least some specified amount of resources allocatable at a given priority.
// For example, all nodes of nodeType "foo" with at least 2 cores and 1Gi memory allocatable at priority 2.
// Nodes are returned in sorted order, from least to most of the specified resource available.
type NodeTypeIterator struct {
	txn *memdb.Txn
	// Only yield nodes of this nodeType.
	nodeTypeId uint64
	// Priority at which to consider allocatable resources on the node.
	priority int32
	// Used to index into node.keys to assert that keys are always increasing.
	// This to detect if the iterator gets stuck.
	// TODO(albin): With better testing we should be able to remove this.
	keyIndex int
	// Name of the memdb index used for node iteration.
	// Should correspond to the priority set for this iterator.
	indexName string
	// NodeDb indexed resources.
	indexedResources []string
	// Pod requests for indexed resources in the same order as indexedResources.
	indexedResourceRequests []int64
	// The resolution with which indexed resources are tracked.
	// In the same order as indexedResources/indexedResourceRequests above.
	// In the same units as indexedResourceRequests above.
	indexedResourceResolution []int64
	// Current lower bound on node allocatable resources looked for.
	// Updated in-place as the iterator makes progress.
	lowerBound []int64
	// Tentative lower-bound.
	newLowerBound []int64
	// memdb key computed from nodeTypeId and lowerBound.
	// Stored here to avoid dynamic allocs.
	key []byte
	// Key for newLowerBound.
	newKey []byte
	// Current iterator into the underlying memdb.
	// Updated in-place whenever lowerBound changes.
	memdbIterator memdb.ResultIterator
	// Used to detect if the iterator gets stuck in a loop.
	previousKey  []byte
	previousNode *internaltypes.Node
}

func NewNodeTypeIterator(
	txn *memdb.Txn,
	nodeTypeId uint64,
	indexName string,
	priority int32,
	keyIndex int,
	indexedResources []string,
	indexedResourceRequests []int64,
	indexedResourceResolution []int64,
) (*NodeTypeIterator, error) {
	if len(indexedResources) != len(indexedResourceRequests) {
		return nil, errors.Errorf("indexedResources and resourceRequirements are not of equal length")
	}
	if len(indexedResources) != len(indexedResourceResolution) {
		return nil, errors.Errorf("indexedResources and indexedResourceResolution are not of equal length")
	}
	if keyIndex < 0 {
		return nil, errors.Errorf("keyIndex is negative: %d", keyIndex)
	}
	it := &NodeTypeIterator{
		txn:                       txn,
		nodeTypeId:                nodeTypeId,
		priority:                  priority,
		keyIndex:                  keyIndex,
		indexName:                 indexName,
		indexedResources:          indexedResources,
		indexedResourceRequests:   indexedResourceRequests,
		indexedResourceResolution: indexedResourceResolution,
		lowerBound:                slices.Clone(indexedResourceRequests),
		newLowerBound:             slices.Clone(indexedResourceRequests),
	}
	memdbIt, err := it.newNodeTypeIterator()
	if err != nil {
		return nil, err
	}
	it.memdbIterator = memdbIt
	return it, nil
}

func (it *NodeTypeIterator) newNodeTypeIterator() (memdb.ResultIterator, error) {
	// TODO(albin): We're re-computing the key unnecessarily here.
	it.key = NodeIndexKey(it.key[0:0], it.nodeTypeId, it.lowerBound)
	memdbIt, err := it.txn.LowerBound(
		"nodes",
		it.indexName,
		it.key,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return memdbIt, nil
}

func (it *NodeTypeIterator) WatchCh() <-chan struct{} {
	panic("not implemented")
}

func (it *NodeTypeIterator) Next() interface{} {
	v, err := it.NextNode()
	if err != nil {
		panic(err)
	}
	return v
}

func (it *NodeTypeIterator) NextNode() (*internaltypes.Node, error) {
	for {
		v := it.memdbIterator.Next()
		if v == nil {
			return nil, nil
		}
		node := v.(*internaltypes.Node)
		if it.keyIndex >= len(node.Keys) {
			return nil, errors.Errorf("keyIndex is %d, but node %s has only %d keys", it.keyIndex, node.GetId(), len(node.Keys))
		}
		nodeKey := node.Keys[it.keyIndex]
		if it.previousKey != nil && bytes.Compare(it.previousKey, nodeKey) != -1 {
			return nil, errors.Errorf(
				"iteration loop detected: key %x of node %#v is not greater than key %x of node %#v",
				nodeKey, node, it.previousKey, it.previousNode,
			)
		}
		it.previousKey = nodeKey
		it.previousNode = node
		if node.GetNodeTypeId() != it.nodeTypeId {
			// There are no more nodes of this nodeType.
			return nil, nil
		}
		allocatableByPriority := node.AllocatableByPriority[it.priority]
		if allocatableByPriority.IsEmpty() {
			return nil, errors.Errorf("node %s has no resources registered at priority %d: %v", node.GetId(), it.priority, node.AllocatableByPriority)
		}
		for i, t := range it.indexedResources {
			nodeQuantity := allocatableByPriority.GetByNameZeroIfMissing(t)
			requestQuantity := it.indexedResourceRequests[i]
			it.newLowerBound[i] = roundQuantityToResolution(nodeQuantity, it.indexedResourceResolution[i])

			// If nodeQuantity < requestQuantity, replace the iterator using the lowerBound.
			// If nodeQuantity >= requestQuantity for all resources, return the node.
			if nodeQuantity < requestQuantity {
				for j := i; j < len(it.indexedResources); j++ {
					it.newLowerBound[j] = it.indexedResourceRequests[j]
				}

				it.newKey = NodeIndexKey(it.newKey[0:0], it.nodeTypeId, it.newLowerBound)
				if bytes.Compare(it.key, it.newKey) == -1 {
					// TODO(albin): Temporary workaround. Shouldn't be necessary.
					lowerBound := it.lowerBound
					it.lowerBound = it.newLowerBound
					it.newLowerBound = lowerBound
				} else {
					log.Warnf(
						"new lower-bound %x is not greater than current bound %x",
						it.newKey, it.key,
					)
					break
				}

				memdbIterator, err := it.newNodeTypeIterator()
				if err != nil {
					return nil, err
				}
				it.memdbIterator = memdbIterator
				break
			} else if i == len(it.indexedResources)-1 {
				return node, nil
			}
		}
	}
}
