package compute

const (
	UnknownCommandID = iota
	SetCommandID
	GetCommandID
	DelCommandID
)

type CommandSpec struct {
	ID       int
	ArgCount int
}

var CommandSpecList = map[string]CommandSpec{
	"SET": {ID: SetCommandID, ArgCount: 2},
	"GET": {ID: GetCommandID, ArgCount: 1},
	"DEL": {ID: DelCommandID, ArgCount: 1},
}
