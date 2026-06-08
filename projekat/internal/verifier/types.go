package verifier

const LayerStructuralAV = "structural_av"
const LayerClamAV = "clamav"

const (
	StatusVerified = "verified"
	StatusRejected = "rejected"
)

// Finding is a single policy violation reported by a verifier layer.
type Finding struct {
	Rule    string `json:"rule"`
	File    string `json:"file,omitempty"`
	Message string `json:"message"`
	Layer   string `json:"layer"`
}

// LayerResult is the outcome of one verifier layer.
type LayerResult struct {
	Name     string    `json:"name"`
	OK       bool      `json:"ok"`
	Skipped  bool      `json:"skipped,omitempty"`
	Findings []Finding `json:"findings,omitempty"`
}

// Result is the full verifier outcome.
type Result struct {
	OK      bool          `json:"ok"`
	Status  string        `json:"status"`
	WorkDir string        `json:"work_dir,omitempty"`
	Layers  []LayerResult `json:"layers"`
}
