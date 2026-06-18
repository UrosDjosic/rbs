package verifier

// Options tune verifier limits. Nil uses DefaultOptions.
type Options struct {
	MaxFiles             int
	MaxUncompressedBytes int64
	MaxFileBytes         int64
	SkipBandit           bool
	BanditPath           string // optional override, e.g. C:\Python\Scripts\bandit.exe
	BanditMinSeverity    string // LOW, MEDIUM, HIGH (default MEDIUM)
	SkipClamAV           bool
	ClamScanPath         string // optional override, e.g. C:\Program Files\ClamAV\clamscan.exe
	ClamAVDatabaseDir    string // virus DB dir for clamscan -d (default: storage/clamav/database)
	SkipPipAudit         bool
	PipAuditPath         string // optional override for pip-audit binary
}

func DefaultOptions() Options {
	return Options{
		MaxFiles:             20,
		MaxUncompressedBytes: 10 << 20, // 10 MiB
		MaxFileBytes:         1 << 20,  // 1 MiB per file
		BanditMinSeverity:    "LOW",
	}
}

func (o *Options) normalized() Options {
	def := DefaultOptions()
	if o == nil {
		return def
	}
	out := *o
	if out.MaxFiles <= 0 {
		out.MaxFiles = def.MaxFiles
	}
	if out.MaxUncompressedBytes <= 0 {
		out.MaxUncompressedBytes = def.MaxUncompressedBytes
	}
	if out.MaxFileBytes <= 0 {
		out.MaxFileBytes = def.MaxFileBytes
	}
	if out.BanditMinSeverity == "" {
		out.BanditMinSeverity = def.BanditMinSeverity
	}
	return out
}
