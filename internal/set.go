package internal

type Set map[string]bool

func (s Set) Add(element string) {
	s[element] = true
}

func (s Set) Remove(element string) {
	delete(s, element)
}

func (s Set) Contains(element string) bool {
	_, exists := s[element]
	return exists
}

func (s Set) Elements() []string {
	elements := make([]string, 0, len(s))
	for key := range s {
		elements = append(elements, key)
	}
	return elements
}
