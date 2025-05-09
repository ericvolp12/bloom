package bloom

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// This implementation of Bloom filters _is_
// safe for concurrent use!!!
func TestConcurrent(t *testing.T) {
	gmp := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(gmp)

	f := New(1000, 4)
	n1 := []byte("Bess")
	n2 := []byte("Jane")
	f.Add(n1)
	f.Add(n2)

	var wg sync.WaitGroup
	const try = 1000
	var err1, err2 error

	wg.Add(1)
	go func() {
		for i := 0; i < try; i++ {
			n1b := f.Test(n1)
			if !n1b {
				err1 = fmt.Errorf("%v should be in", n1)
				break
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for i := 0; i < try; i++ {
			n2b := f.Test(n2)
			if !n2b {
				err2 = fmt.Errorf("%v should be in", n2)
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()

	if err1 != nil {
		t.Fatal(err1)
	}
	if err2 != nil {
		t.Fatal(err2)
	}
}

func TestBasic(t *testing.T) {
	f := New(1000, 4)
	n1 := []byte("Bess")
	n2 := []byte("Jane")
	n3 := []byte("Emma")
	f.Add(n1)
	n3a := f.TestAndAdd(n3)
	n1b := f.Test(n1)
	n2b := f.Test(n2)
	n3b := f.Test(n3)
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
}

func TestBasicUint32(t *testing.T) {
	f := New(1000, 4)
	n1 := make([]byte, 4)
	n2 := make([]byte, 4)
	n3 := make([]byte, 4)
	n4 := make([]byte, 4)
	n5 := make([]byte, 4)
	binary.BigEndian.PutUint32(n1, 100)
	binary.BigEndian.PutUint32(n2, 101)
	binary.BigEndian.PutUint32(n3, 102)
	binary.BigEndian.PutUint32(n4, 103)
	binary.BigEndian.PutUint32(n5, 104)
	f.Add(n1)
	n3a := f.TestAndAdd(n3)
	n1b := f.Test(n1)
	n2b := f.Test(n2)
	n3b := f.Test(n3)
	n5a := f.TestOrAdd(n5)
	n5b := f.Test(n5)
	f.Test(n4)
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
	if n5a {
		t.Errorf("%v should not be in the first time we look.", n5)
	}
	if !n5b {
		t.Errorf("%v should be in the second time we look.", n5)
	}
}

func TestNewWithLowNumbers(t *testing.T) {
	f := New(0, 0)
	if f.k != 1 {
		t.Errorf("%v should be 1", f.k)
	}
	if f.m != 1 {
		t.Errorf("%v should be 1", f.m)
	}
}

func TestString(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)
	n1 := "Love"
	n2 := "is"
	n3 := "in"
	n4 := "bloom"
	n5 := "blooms"
	f.AddString(n1)
	n3a := f.TestAndAddString(n3)
	n1b := f.TestString(n1)
	n2b := f.TestString(n2)
	n3b := f.TestString(n3)
	n5a := f.TestOrAddString(n5)
	n5b := f.TestString(n5)
	f.TestString(n4)
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
	if n5a {
		t.Errorf("%v should not be in the first time we look.", n5)
	}
	if !n5b {
		t.Errorf("%v should be in the second time we look.", n5)
	}

}

func testEstimated(n uint, maxFp float64, t *testing.T) {
	m, k := EstimateParameters(n, maxFp)
	fpRate := EstimateFalsePositiveRate(m, k, n)
	if fpRate > 1.5*maxFp {
		t.Errorf("False positive rate too high: n: %v; m: %v; k: %v; maxFp: %f; fpRate: %f, fpRate/maxFp: %f", n, m, k, maxFp, fpRate, fpRate/maxFp)
	}
}

func TestEstimated1000_0001(t *testing.T)   { testEstimated(1000, 0.000100, t) }
func TestEstimated10000_0001(t *testing.T)  { testEstimated(10000, 0.000100, t) }
func TestEstimated100000_0001(t *testing.T) { testEstimated(100000, 0.000100, t) }

func TestEstimated1000_001(t *testing.T)   { testEstimated(1000, 0.001000, t) }
func TestEstimated10000_001(t *testing.T)  { testEstimated(10000, 0.001000, t) }
func TestEstimated100000_001(t *testing.T) { testEstimated(100000, 0.001000, t) }

func TestEstimated1000_01(t *testing.T)   { testEstimated(1000, 0.010000, t) }
func TestEstimated10000_01(t *testing.T)  { testEstimated(10000, 0.010000, t) }
func TestEstimated100000_01(t *testing.T) { testEstimated(100000, 0.010000, t) }

func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// The following function courtesy of Nick @turgon
// This helper function ranges over the input data, applying the hashing
// which returns the bit locations to set in the filter.
// For each location, increment a counter for that bit address.
//
// If the Bloom Filter's location() method distributes locations uniformly
// at random, a property it should inherit from its hash function, then
// each bit location in the filter should end up with roughly the same
// number of hits.  Importantly, the value of k should not matter.
//
// Once the results are collected, we can run a chi squared goodness of fit
// test, comparing the result histogram with the uniform distribition.
// This yields a test statistic with degrees-of-freedom of m-1.
func chiTestBloom(m, k, rounds uint, elements [][]byte) (succeeds bool) {
	f := New(m, k)
	results := make([]uint, m)
	chi := make([]float64, m)

	for _, data := range elements {
		h := baseHashes(data)
		for i := uint(0); i < f.k; i++ {
			results[f.location(h, i)]++
		}
	}

	// Each element of results should contain the same value: k * rounds / m.
	// Let's run a chi-square goodness of fit and see how it fares.
	var chiStatistic float64
	e := float64(k*rounds) / float64(m)
	for i := uint(0); i < m; i++ {
		chi[i] = math.Pow(float64(results[i])-e, 2.0) / e
		chiStatistic += chi[i]
	}

	// this tests at significant level 0.005 up to 20 degrees of freedom
	table := [20]float64{
		7.879, 10.597, 12.838, 14.86, 16.75, 18.548, 20.278,
		21.955, 23.589, 25.188, 26.757, 28.3, 29.819, 31.319, 32.801, 34.267,
		35.718, 37.156, 38.582, 39.997}
	df := min(m-1, 20)

	succeeds = table[df-1] > chiStatistic
	return

}

func TestLocation(t *testing.T) {
	var m, k, rounds uint

	m = 8
	k = 3

	rounds = 100000 // 15000000

	elements := make([][]byte, rounds)

	for x := uint(0); x < rounds; x++ {
		ctrlist := make([]uint8, 4)
		ctrlist[0] = uint8(x)
		ctrlist[1] = uint8(x >> 8)
		ctrlist[2] = uint8(x >> 16)
		ctrlist[3] = uint8(x >> 24)
		data := []byte(ctrlist)
		elements[x] = data
	}

	succeeds := chiTestBloom(m, k, rounds, elements)
	if !succeeds {
		t.Error("random assignment is too unrandom")
	}

}

func TestCap(t *testing.T) {
	f := New(1000, 4)
	if f.Cap() != f.m {
		t.Error("not accessing Cap() correctly")
	}
}

func TestK(t *testing.T) {
	f := New(1000, 4)
	if f.K() != f.k {
		t.Error("not accessing K() correctly")
	}
}

func TestMarshalUnmarshalJSON(t *testing.T) {
	f := New(1000, 4)
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(string(data))

	var g BloomFilter
	err = json.Unmarshal(data, &g)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
}

func TestMarshalUnmarshalJSONValue(t *testing.T) {
	f := BloomFilter{1000, 4, newAtomicBitSet(1000)}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(string(data))

	var g BloomFilter
	err = json.Unmarshal(data, &g)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	data := []byte("{invalid}")

	var g BloomFilter
	err := g.UnmarshalJSON(data)
	if err == nil {
		t.Error("expected error while unmarshalling invalid data")
	}
}

func TestWriteToReadFrom(t *testing.T) {
	var b bytes.Buffer
	f := New(1000, 4)
	_, err := f.WriteTo(&b)
	if err != nil {
		t.Fatal(err)
	}

	g := New(1000, 1)
	_, err = g.ReadFrom(&b)
	if err != nil {
		t.Fatal(err)
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}

	g.Test([]byte(""))
}

func TestReadWriteBinary(t *testing.T) {
	f := New(1000, 4)
	var buf bytes.Buffer
	bytesWritten, err := f.WriteTo(&buf)
	if err != nil {
		t.Fatal(err.Error())
	}
	if bytesWritten != int64(buf.Len()) {
		t.Errorf("incorrect write length %d != %d", bytesWritten, buf.Len())
	}

	var g BloomFilter
	bytesRead, err := g.ReadFrom(&buf)
	if err != nil {
		t.Fatal(err.Error())
	}
	if bytesRead != bytesWritten {
		t.Errorf("read unexpected number of bytes %d != %d", bytesRead, bytesWritten)
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
}

func TestEncodeDecodeGob(t *testing.T) {
	f := New(1000, 4)
	f.Add([]byte("one"))
	f.Add([]byte("two"))
	f.Add([]byte("three"))
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(f)
	if err != nil {
		t.Fatal(err.Error())
	}

	var g BloomFilter
	err = gob.NewDecoder(&buf).Decode(&g)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
	if !g.Test([]byte("three")) {
		t.Errorf("missing value 'three'")
	}
	if !g.Test([]byte("two")) {
		t.Errorf("missing value 'two'")
	}
	if !g.Test([]byte("one")) {
		t.Errorf("missing value 'one'")
	}
}

func TestEqual(t *testing.T) {
	f := New(1000, 4)
	f1 := New(1000, 4)
	g := New(1000, 20)
	h := New(10, 20)
	n1 := []byte("Bess")
	f1.Add(n1)
	if !f.Equal(f) {
		t.Errorf("%v should be equal to itself", f)
	}
	if f.Equal(f1) {
		t.Errorf("%v should not be equal to %v", f, f1)
	}
	if f.Equal(g) {
		t.Errorf("%v should not be equal to %v", f, g)
	}
	if f.Equal(h) {
		t.Errorf("%v should not be equal to %v", f, h)
	}
}

func BenchmarkEstimated(b *testing.B) {
	for n := uint(100000); n <= 100000; n *= 10 {
		for fp := 0.1; fp >= 0.0001; fp /= 10.0 {
			f := NewWithEstimates(n, fp)
			EstimateFalsePositiveRate(f.m, f.k, n)
		}
	}
}

func BenchmarkSeparateTestAndAdd(b *testing.B) {
	f := NewWithEstimates(uint(b.N), 0.0001)
	key := make([]byte, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint32(key, uint32(i))
		f.Test(key)
		f.Add(key)
	}
}

func BenchmarkCombinedTestAndAdd(b *testing.B) {
	f := NewWithEstimates(uint(b.N), 0.0001)
	key := make([]byte, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint32(key, uint32(i))
		f.TestAndAdd(key)
	}
}

func TestMerge(t *testing.T) {
	f := New(1000, 4)
	n1 := []byte("f")
	f.Add(n1)

	g := New(1000, 4)
	n2 := []byte("g")
	g.Add(n2)

	h := New(999, 4)
	n3 := []byte("h")
	h.Add(n3)

	j := New(1000, 5)
	n4 := []byte("j")
	j.Add(n4)

	err := f.Merge(g)
	if err != nil {
		t.Errorf("There should be no error when merging two similar filters")
	}

	err = f.Merge(h)
	if err == nil {
		t.Errorf("There should be an error when merging filters with mismatched m")
	}

	err = f.Merge(j)
	if err == nil {
		t.Errorf("There should be an error when merging filters with mismatched k")
	}

	n2b := f.Test(n2)
	if !n2b {
		t.Errorf("The value doesn't exist after a valid merge")
	}

	n3b := f.Test(n3)
	if n3b {
		t.Errorf("The value exists after an invalid merge")
	}

	n4b := f.Test(n4)
	if n4b {
		t.Errorf("The value exists after an invalid merge")
	}
}

func TestCopy(t *testing.T) {
	f := New(1000, 4)
	n1 := []byte("f")
	f.Add(n1)

	// copy here instead of New
	g := f.Copy()
	n2 := []byte("g")
	g.Add(n2)

	n1fb := f.Test(n1)
	if !n1fb {
		t.Errorf("The value doesn't exist in original after making a copy")
	}

	n1gb := g.Test(n1)
	if !n1gb {
		t.Errorf("The value doesn't exist in the copy")
	}

	n2fb := f.Test(n2)
	if n2fb {
		t.Errorf("The value exists in the original, it should only exist in copy")
	}

	n2gb := g.Test(n2)
	if !n2gb {
		t.Errorf("The value doesn't exist in copy after Add()")
	}
}

func TestTestLocations(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)
	n1 := []byte("Love")
	n2 := []byte("is")
	n3 := []byte("in")
	n4 := []byte("bloom")
	f.Add(n1)
	n3a := f.TestLocations(Locations(n3, f.K()))
	f.Add(n3)
	n1b := f.TestLocations(Locations(n1, f.K()))
	n2b := f.TestLocations(Locations(n2, f.K()))
	n3b := f.TestLocations(Locations(n3, f.K()))
	n4b := f.TestLocations(Locations(n4, f.K()))
	if !n1b {
		t.Errorf("%v should be in.", n1)
	}
	if n2b {
		t.Errorf("%v should not be in.", n2)
	}
	if n3a {
		t.Errorf("%v should not be in the first time we look.", n3)
	}
	if !n3b {
		t.Errorf("%v should be in the second time we look.", n3)
	}
	if n4b {
		t.Errorf("%v should be in.", n4)
	}
}

func TestApproximatedSize(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)
	f.Add([]byte("Love"))
	f.Add([]byte("is"))
	f.Add([]byte("in"))
	f.Add([]byte("bloom"))
	size := f.ApproximatedSize()
	if size != 4 {
		t.Errorf("%d should equal 4.", size)
	}
}

func TestFPP(t *testing.T) {
	f := NewWithEstimates(1000, 0.001)
	for i := uint32(0); i < 1000; i++ {
		n := make([]byte, 4)
		binary.BigEndian.PutUint32(n, i)
		f.Add(n)
	}
	count := 0

	for i := uint32(0); i < 1000; i++ {
		n := make([]byte, 4)
		binary.BigEndian.PutUint32(n, i+1000)
		if f.Test(n) {
			count += 1
		}
	}
	if float64(count)/1000.0 > 0.001 {
		t.Errorf("Excessive fpp")
	}
}

func TestEncodeDecodeBinary(t *testing.T) {
	f := New(1000, 4)
	f.Add([]byte("one"))
	f.Add([]byte("two"))
	f.Add([]byte("three"))
	data, err := f.MarshalBinary()
	if err != nil {
		t.Fatal(err.Error())
	}

	var g BloomFilter
	err = g.UnmarshalBinary(data)
	if err != nil {
		t.Fatal(err.Error())
	}
	if g.m != f.m {
		t.Error("invalid m value")
	}
	if g.k != f.k {
		t.Error("invalid k value")
	}
	if g.b == nil {
		t.Fatal("bitset is nil")
	}
	if !g.b.Equal(f.b) {
		t.Error("bitsets are not equal")
	}
	if !g.Test([]byte("three")) {
		t.Errorf("missing value 'three'")
	}
	if !g.Test([]byte("two")) {
		t.Errorf("missing value 'two'")
	}
	if !g.Test([]byte("one")) {
		t.Errorf("missing value 'one'")
	}
}

func BenchmarkAdd(b *testing.B) {
	f := NewWithEstimates(uint(b.N), 0.0001)

	// Generate N random key values
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, 100)
		binary.BigEndian.PutUint32(key, uint32(i))
		keys[i] = key
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Add(keys[i])
	}
}
func BenchmarkTest(b *testing.B) {
	f := NewWithEstimates(uint(b.N), 0.0001)

	// Generate N random key values
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, 100)
		binary.BigEndian.PutUint32(key, uint32(i))
		keys[i] = key
	}

	// Add keys to the filter
	for i := 0; i < b.N; i++ {
		f.Add(keys[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Test(keys[i])
	}
}

func BenchmarkConcurrent(b *testing.B) {
	gmp := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(gmp)

	f := NewWithEstimates(uint(b.N), 0.0001)
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, 100)
		binary.BigEndian.PutUint32(key, uint32(i))
		keys[i] = key
	}

	var wg sync.WaitGroup
	wg.Add(2)

	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			f.Add(keys[i])
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < b.N; i++ {
			f.Test(keys[i])
		}
		wg.Done()
	}()

	wg.Wait()
}

func BenchmarkHighConcurrency(b *testing.B) {
	gmp := runtime.GOMAXPROCS(32)
	defer runtime.GOMAXPROCS(gmp)

	f := NewWithEstimates(uint(b.N), 0.0001)
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		key := make([]byte, 100)
		binary.BigEndian.PutUint32(key, uint32(i))
		keys[i] = key
	}

	var wg sync.WaitGroup
	wg.Add(32)

	b.ResetTimer()

	for i := 0; i < 32; i++ {
		go func(i int) {
			for j := 0; j < b.N; j++ {
				if i%2 == 0 {
					f.Add(keys[j])
				} else {
					f.Test(keys[j])
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}

// TestConcurrentAddAndTestThroughput tests the throughput of concurrent
// add and test operations on the Bloom filter.
// It adds 10,000 keys per second in a single goroutine
// It tests as many keys as possible in 7 other goroutines
// It completes when it has run for 10 seconds
// On my machine:
// Total keys tested: 139936012, Throughput: 13993601.20000 keys/sec (nearly 14m tests/sec)
func TestConcurrentAddAndTestThroughput(t *testing.T) {
	gmp := runtime.GOMAXPROCS(8)
	defer runtime.GOMAXPROCS(gmp)

	filterSize := 10_000_000
	f := NewWithEstimates(uint(filterSize), 0.0001)
	keys := make([][]byte, filterSize/10)
	for i := 0; i < filterSize/10; i++ {
		key := make([]byte, 100)
		binary.BigEndian.PutUint32(key, uint32(i))
		keys[i] = key
	}

	addRate := rate.NewLimiter(rate.Limit(10000), 10000) // 10,000 adds per second

	var wg sync.WaitGroup
	wg.Add(8)

	startTime := time.Now()
	endTime := startTime.Add(10 * time.Second)

	ctx := context.Background()

	go func() {
		for time.Now().Before(endTime) {
			if err := addRate.Wait(ctx); err != nil {
				return
			}
			for i := 0; i < len(keys); i++ {
				f.Add(keys[i])
			}
		}
		wg.Done()
	}()

	totalKeysTested := atomic.Int64{}

	for i := 0; i < 7; i++ {
		go func() {
			keysTested := 0
			for time.Now().Before(endTime) {
				// Start at a random position in the keys slice
				startIdx := rand.Intn(len(keys))
				for j := startIdx; j < len(keys); j++ {
					f.Test(keys[j])
					keysTested++
				}
			}
			totalKeysTested.Add(int64(keysTested))
			wg.Done()
		}()
	}

	wg.Wait()

	totalKeys := totalKeysTested.Load()
	if totalKeys > 0 {
		throughput := float64(totalKeys) / 10.0 // 10 seconds
		t.Errorf("Total keys tested: %d, Throughput: %.5f keys/sec", totalKeys, throughput)
	}
}
