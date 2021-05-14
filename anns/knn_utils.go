package anns

import (
	"errors"
	"math"
	"sort"

	"github.com/sachaservan/vec"
)

// BruteForceSearchNN enumerates all points and finds the closest point (according to provided distance function)
// to the query point
func BruteForceSearchNN(query *vec.Vec, points []*vec.Vec, distance DistanceFunction) *vec.Vec {
	minDistance := math.MaxFloat64
	minIndex := 0 // index of the point with minDistance
	for i := 0; i < len(points); i++ {
		dist := distance(query, points[i])
		if dist < minDistance {
			minDistance = dist
			minIndex = i
		}

		// found a perfect match
		if dist == 0 {
			break
		}
	}

	return points[minIndex]
}

// BruteForceSearchTopKNN as BruteForceSearchNN but returns the k nearest neighbors
func BruteForceSearchTopKNN(query *vec.Vec, points []*vec.Vec, ids []int, distance DistanceFunction, k int) []*vec.Vec {

	seen := make(map[int]bool)

	dists := make([]tuple, 0)
	for i := 0; i < len(points); i++ {

		// prevent duplicates
		if contains, _ := seen[ids[i]]; contains {
			continue
		}
		seen[ids[i]] = true

		dist := distance(query, points[i])
		dists = append(dists, tuple{a: i, b: int(dist * 1000000)}) // TODO: ugly scale factor
	}

	sort.Sort(tupleArr(dists))

	sortedCandidates := make([]*vec.Vec, 0)
	for _, tuple := range dists {
		sortedCandidates = append(sortedCandidates, points[tuple.a])
	}

	if k >= len(sortedCandidates) {
		return sortedCandidates
	}

	return sortedCandidates[:k]
}

// GetMajorityCandidate return the candidate that is represented by the majority of ids
// (if such a candidate exists)
func GetMajorityCandidate(candidates []*vec.Vec, ids []int) (*vec.Vec, error) {

	if len(candidates) == 0 {
		return nil, errors.New("no candidates provided")
	}

	sortedCandidates, _ := GetSortedCandidates(candidates, ids)

	return sortedCandidates[0], nil
}

type tuple struct {
	a int
	b int
}

type tupleArr []tuple

func (a tupleArr) Len() int           { return len(a) }
func (a tupleArr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tupleArr) Less(i, j int) bool { return a[i].b < a[j].b }

// GetSortedCandidates return the candidates sorted according to frequency with which they appear
// high to low in the list; the frequency is determined by examining ids
func GetSortedCandidates(candidates []*vec.Vec, ids []int) ([]*vec.Vec, error) {

	if len(candidates) == 0 {
		return nil, errors.New("no candidates provided")
	}

	indexMap := make(map[int]int, len(ids))
	for _, index := range ids {
		indexMap[index]++
	}

	// extract unique candidates indices from the indexMap
	sorted := make([]tuple, 0)
	for k, v := range indexMap {
		for i, id := range ids {
			if k == id {
				sorted = append(sorted, tuple{a: i, b: v})
				break
			}
		}
	}

	sort.Sort(sort.Reverse(tupleArr(sorted)))

	sortedCandidates := make([]*vec.Vec, 0)
	for _, sort := range sorted {
		sortedCandidates = append(sortedCandidates, candidates[sort.a])
	}

	return sortedCandidates, nil
}
