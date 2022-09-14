/**
 * @Author raven
 * @Description
 * @Date 2022/9/13
 **/
package utils

import (
	"github.com/RavenHuo/go-kit/collections/treemaps"
	"math/rand"
)

type weightRandomMap struct {
	weightMaps *treemaps.TreeMap
}

func NewWeightRandomMap(data map[interface{}]float64) *weightRandomMap {
	weightMap := treemaps.New(func(a treemaps.Key, b treemaps.Key) bool {
		return a.(float64) < b.(float64)
	})
	for k, v := range data {
		weight := v
		if weight <= 0 {
			continue
		}
		var lastWeight float64
		if weightMap.Len() != 0 {
			lastWeight = weightMap.LastKey().(float64)
		}
		weightMap.Set(treemaps.Value(weight+lastWeight), treemaps.Value(k))
	}
	return &weightRandomMap{weightMaps: weightMap}
}

//             double randomWeight = weightMap.lastKey() * Math.random();
//            SortedMap<Double, K> tailMap = weightMap.tailMap(randomWeight, false);
func (m *weightRandomMap) Random() treemaps.Value {
	// 获取一个随机权重
	randomWeight := float64(rand.Intn(int(m.weightMaps.LastKey().(float64))))
	// 获取一个比随机权重大的 kv
	iter := m.weightMaps.LowerBound(treemaps.Value(randomWeight))
	// 返回第一个
	return iter.Value()
}
