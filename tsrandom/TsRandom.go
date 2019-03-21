package tsrandom

type TsRandom struct {
	mag01        [2]uint
	mt           [624]uint
	mti          int
}
var N = 624
var M = 397
var MATRIX_A = uint(0x9908b0df)
var UPPER_MASK = uint(0x80000000)
var LOWER_MASK = uint(0x7fffffff)
var MAX_RAND_INT = int(0x7fffffff)

func New(seed int) TsRandom {
	ts := TsRandom{
		mag01:        [2]uint{uint(0x0), uint(0x9908b0df)},
		mti:          625,
	}
	ts.init_genrand(uint(seed))
	return ts
}

func (ts *TsRandom) init_genrand(seed uint) {
	ts.mt[0] = seed & uint(0xffffffff)
	for ts.mti = 1; ts.mti < N; ts.mti++ {
		ts.mt[ts.mti] = uint(1812433253)*(ts.mt[ts.mti-1]^(ts.mt[ts.mti-1]>>30)) + uint(ts.mti)
		ts.mt[ts.mti] &= uint(0xffffffff)
	}
}
func (ts *TsRandom)Next() int{
	return ts.genrand_int31()
}
func (ts *TsRandom)RangeInt(minValue int,maxValue int) int{
	if minValue > maxValue {
		tmp := maxValue
		maxValue = minValue
		minValue = tmp
	}
	rangeV := maxValue - minValue
	return minValue + ts.Next() % rangeV
}
func (ts *TsRandom)RandomBool() bool{
	value := ts.RangeInt(0, 2)
	if value == 0{
		return false
	}else{
		return true
	}
}
func (ts *TsRandom)genrand_int31()int{
	return int(ts.genrand_int32() >> 1)
}
func (ts *TsRandom)genrand_int32() uint{
	var y uint
	if ts.mti >= N {
		var kk int
		if ts.mti == N + 1 {
			ts.init_genrand(5489)
		}
			for kk = 0; kk < N- M; kk++ {
			y = (ts.mt[kk] & UPPER_MASK) | (ts.mt[kk + 1] & LOWER_MASK)
			ts.mt[kk] = ts.mt[kk + M] ^ (y >> 1) ^ ts.mag01[y & uint(0x1)]
			}
			for ; kk < N-1; kk++ {
			y = (ts.mt[kk] & UPPER_MASK) | (ts.mt[kk + 1] & LOWER_MASK)
				ts.mt[kk] = ts.mt[kk + (M - N)] ^ (y >> 1) ^ ts.mag01[y & uint(0x1)]
			}
			y = (ts.mt[N-1] & UPPER_MASK) | (ts.mt[0] & LOWER_MASK)
		ts.mt[N-1] = ts.mt[M-1] ^ (y >> 1) ^ ts.mag01[y & uint(0x1)]
		ts.mti = 0
	}
	y = ts.mt[ts.mti]
	ts.mti++
	y ^= y >> 11
	y ^= (y << 7) & uint(0x9d2c5680)
	y ^= (y << 15) & uint(0xefc60000)
	y ^= y >> 18
	return y
}
