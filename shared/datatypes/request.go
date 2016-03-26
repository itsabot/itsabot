package dt

// Request for Abot to perform some command.
type Request struct {
	CMD        string     `json:"cmd"`
	UserID     uint64     `json:"uid"`
	FlexID     string     `json:"flexid"`
	FlexIDType FlexIDType `json:"flexidtype"`
}
