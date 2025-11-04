package entropy

type ANSState struct {
	State    int32
	HasState bool
}

func NewANSState() *ANSState {
	s := &ANSState{}
	return s
}

func (s *ANSState) SetState(state int32) {
	s.State = state
	s.HasState = true
}
