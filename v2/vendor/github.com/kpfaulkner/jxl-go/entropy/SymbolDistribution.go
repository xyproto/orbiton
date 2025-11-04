package entropy

import "github.com/kpfaulkner/jxl-go/jxlio"

type SymbolDistribution interface {
	ReadSymbol(reader jxlio.BitReader, state *ANSState) (int32, error)
	SetConfig(config *HybridIntegerConfig)
	GetConfig() *HybridIntegerConfig
}

type SymbolDistributionBase struct {
	config          *HybridIntegerConfig
	logBucketSize   int32
	alphabetSize    int32
	logAlphabetSize int32
}

func NewSymbolDistributionBase() *SymbolDistributionBase {
	rcvr := &SymbolDistributionBase{}
	return rcvr
}

func (rcvr *SymbolDistributionBase) ReadSymbol(reader jxlio.BitReader, state *ANSState) (int32, error) {

	return 0, nil
}

func (rcvr *SymbolDistributionBase) SetConfig(config *HybridIntegerConfig) {
	rcvr.config = config
}

func (rcvr *SymbolDistributionBase) GetConfig() *HybridIntegerConfig {
	return rcvr.config
}
