package mapping

import "fmt"

type BidirectionalIndexMap struct {
	uuidToIndex map[string]int
	indexToUUID []string
}

func NewBidirectionalIndexMap(orderedUUIDs []string) (*BidirectionalIndexMap, error) {
	m := &BidirectionalIndexMap{
		uuidToIndex: make(map[string]int, len(orderedUUIDs)),
		indexToUUID: make([]string, 0, len(orderedUUIDs)),
	}

	for _, uuid := range orderedUUIDs {
		if uuid == "" {
			continue
		}
		if _, exists := m.uuidToIndex[uuid]; exists {
			return nil, fmt.Errorf("duplicate uuid in mapping input: %s", uuid)
		}
		m.uuidToIndex[uuid] = len(m.indexToUUID)
		m.indexToUUID = append(m.indexToUUID, uuid)
	}

	return m, nil
}

func (m *BidirectionalIndexMap) IndexOf(uuid string) (int, bool) {
	index, ok := m.uuidToIndex[uuid]
	return index, ok
}

func (m *BidirectionalIndexMap) UUIDOf(index int) (string, bool) {
	if index < 0 || index >= len(m.indexToUUID) {
		return "", false
	}
	return m.indexToUUID[index], true
}

func (m *BidirectionalIndexMap) Ordered() []string {
	out := make([]string, len(m.indexToUUID))
	copy(out, m.indexToUUID)
	return out
}

func (m *BidirectionalIndexMap) Reverse() map[string]int {
	out := make(map[string]int, len(m.uuidToIndex))
	for uuid, index := range m.uuidToIndex {
		out[uuid] = index
	}
	return out
}

func (m *BidirectionalIndexMap) Size() int {
	return len(m.indexToUUID)
}
