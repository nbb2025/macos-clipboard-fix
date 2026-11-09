package core

// NewReadOnly does not create or write application records. The diagnostic CLI
// can therefore run alongside the GUI without overwriting its backup journal.
func NewReadOnly(home string) *Engine {
	return &Engine{Home: home, Runner: CommandRunner{}, readOnly: true,
		history: []Event{}, backups: []Backup{}, verifications: []Verification{}}
}
