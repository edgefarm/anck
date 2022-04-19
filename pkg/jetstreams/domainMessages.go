package jetstreams

// DomainMessagesType is a type for holding messages for a domain
type DomainMessagesType struct {
	OkMap  map[string][]string
	ErrMap map[string][]string
}

// NewDomainMessages returns a new DomainMessagesType
func NewDomainMessages() *DomainMessagesType {
	return &DomainMessagesType{
		OkMap:  make(map[string][]string),
		ErrMap: make(map[string][]string),
	}
}

// Ok adds a message for the positive case
func (de *DomainMessagesType) Ok(domain string, err string) {
	if de.OkMap == nil {
		de.OkMap = make(map[string][]string)
	}
	if _, ok := de.OkMap[domain]; !ok {
		de.OkMap[domain] = make([]string, 0)
	}
	de.OkMap[domain] = append(de.OkMap[domain], err)
}

// Error adds a message for the error case
func (de *DomainMessagesType) Error(domain string, err string) {
	if de.ErrMap == nil {
		de.OkMap = make(map[string][]string)
	}
	if _, ok := de.ErrMap[domain]; !ok {
		de.ErrMap[domain] = make([]string, 0)
	}
	de.ErrMap[domain] = append(de.ErrMap[domain], err)
}
