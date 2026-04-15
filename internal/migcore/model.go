package migcore

type Migration struct {
	Version  string
	Name     string
	FileName string
	SQL      string
	Checksum string
}
