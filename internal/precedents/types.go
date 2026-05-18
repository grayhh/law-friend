package precedents

type Precedent struct {
	ID             string `json:"id"`
	CaseNumber     string `json:"case_number"`
	CaseName       string `json:"case_name"`
	Court          string `json:"court"`
	DecisionDate   string `json:"decision_date"`
	Issues         string `json:"issues"`
	Summary        string `json:"summary"`
	FullText       string `json:"full_text"`
	SourceURL      string `json:"source_url,omitempty"`
	CaseKind       string `json:"case_kind,omitempty"`
	ReferencesLaw  string `json:"references_law,omitempty"`
	ReferencesPrec string `json:"references_prec,omitempty"`
}
