package frame

type XorShiro struct {
	state0   []uint64
	state1   []uint64
	batch    []uint64
	batchPos int
}

func (s *XorShiro) fill(bits []int64) {
	for i := 0; i < len(bits); i += 2 {
		l := s.nextLong()
		bits[i] = int64(l & uint64(0xFF_FF_FF_FF))
		bits[i+1] = int64((l >> 32) & 0xFF_FF_FF_FF)
	}
}

func (s *XorShiro) nextLong() uint64 {
	s.fillBatch()
	b := s.batch[s.batchPos]
	s.batchPos++
	return b
}

func (s *XorShiro) fillBatch() {
	if s.batchPos < len(s.batch) {
		return
	}
	for i := 0; i < len(s.batch); i++ {
		a := s.state1[i]
		b := s.state0[i]
		s.batch[i] = a + b
		s.state0[i] = a
		b ^= b << 23
		s.state1[i] = b ^ a ^ (b >> 18) ^ (a >> 5)
	}
	s.batchPos = 0
}

func NewXorShiroWith4Seeds(seed0 int32, seed1 int32, seed2 int32, seed3 int32) *XorShiro {

	return NewXorShiroWith2Seeds(
		int64(seed0)<<32|int64(seed1)&0xFF_FF_FF_FF,
		int64(seed2)<<32|int64(seed3)&0xFF_FF_FF_FF)
}

func NewXorShiroWith2Seeds(seed0 int64, seed1 int64) *XorShiro {
	xs := &XorShiro{}
	xs.state0 = make([]uint64, 8)
	xs.state1 = make([]uint64, 8)
	xs.batch = make([]uint64, 8)
	xs.batchPos = 0

	xs.state0[0] = splitMix64(uint64(seed0) + 0x9e3779b97f4a7c15)
	xs.state1[0] = splitMix64(uint64(seed1) + 0x9e3779b97f4a7c15)
	for i := 1; i < 8; i++ {
		xs.state0[i] = splitMix64(uint64(xs.state0[i-1]))
		xs.state1[i] = splitMix64(uint64(xs.state1[i-1]))
	}
	return xs
}

func splitMix64(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
