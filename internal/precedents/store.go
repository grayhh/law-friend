package precedents

type Store interface {
	Search(query string, k int) ([]Precedent, error)
	GetByID(id string) (*Precedent, error)
	All() []Precedent
}
