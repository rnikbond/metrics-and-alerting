package reporter

//const (
//	ReportTypeURL = iota + 1
//	ReportTypeJSON
//	ReportTypeBatchJSON
//)

var (
	ReportTypeURL = ReportType{1, ""}
)

type ReportType struct {
	Type int
	URL  string
}
