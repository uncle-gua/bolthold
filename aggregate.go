// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package bolthold

import (
	"fmt"
	"reflect"
	"sort"

	"go.etcd.io/bbolt"
)

// AggregateResult allows you to access the results of an aggregate query
type AggregateResult struct {
	reduction []reflect.Value // always pointers
	group     []reflect.Value
	sortby    string
}

// Group returns the field grouped by in the query
func (a *AggregateResult) Group(result ...interface{}) {
	for i := range result {
		resultVal := reflect.ValueOf(result[i])
		if resultVal.Kind() != reflect.Ptr {
			panic("result argument must be an address")
		}

		if i >= len(a.group) {
			panic(fmt.Sprintf("There is not %d elements in the grouping", i))
		}

		resultVal.Elem().Set(a.group[i])
	}
}

// Reduction is the collection of records that are part of the AggregateResult Group
func (a *AggregateResult) Reduction(result interface{}) {
	resultVal := reflect.ValueOf(result)

	if resultVal.Kind() != reflect.Ptr || resultVal.Elem().Kind() != reflect.Slice {
		panic("result argument must be a slice address")
	}

	sliceVal := resultVal.Elem()

	elType := sliceVal.Type().Elem()

	for i := range a.reduction {
		if elType.Kind() == reflect.Ptr {
			sliceVal = reflect.Append(sliceVal, a.reduction[i])
		} else {
			sliceVal = reflect.Append(sliceVal, a.reduction[i].Elem())
		}
	}

	resultVal.Elem().Set(sliceVal.Slice(0, sliceVal.Len()))
}

type aggregateResultSort AggregateResult

func (a *aggregateResultSort) Len() int { return len(a.reduction) }
func (a *aggregateResultSort) Swap(i, j int) {
	a.reduction[i], a.reduction[j] = a.reduction[j], a.reduction[i]
}
func (a *aggregateResultSort) Less(i, j int) bool {
	//reduction values are always pointers
	iVal, err := fieldValue(a.reduction[i].Elem(), a.sortby)
	if err != nil {
		panic(err)
	}

	jVal, err := fieldValue(a.reduction[j].Elem(), a.sortby)
	if err != nil {
		panic(err)
	}

	c, err := compare(iVal, jVal)
	if err != nil {
		panic(err)
	}

	return c == -1
}

// Sort sorts the aggregate reduction by the passed in field in ascending order
// Sort is called automatically by calls to Min / Max to get the min and max values
func (a *AggregateResult) Sort(field string) {
	if !startsUpper(field) {
		panic("The first letter of a field must be upper-case")
	}
	if a.sortby == field {
		// already sorted
		return
	}

	a.sortby = field
	sort.Sort((*aggregateResultSort)(a))
}

// Max Returns the maxiumum value of the Aggregate Grouping, uses the Comparer interface
func (a *AggregateResult) Max(field string, result interface{}) {
	a.Sort(field)

	resultVal := reflect.ValueOf(result)
	if resultVal.Kind() != reflect.Ptr {
		panic("result argument must be an address")
	}

	if resultVal.IsNil() {
		panic("result argument must not be nil")
	}

	resultVal.Elem().Set(a.reduction[len(a.reduction)-1].Elem())
}

// Min returns the minimum value of the Aggregate Grouping, uses the Comparer interface
func (a *AggregateResult) Min(field string, result interface{}) {
	a.Sort(field)

	resultVal := reflect.ValueOf(result)
	if resultVal.Kind() != reflect.Ptr {
		panic("result argument must be an address")
	}

	if resultVal.IsNil() {
		panic("result argument must not be nil")
	}

	resultVal.Elem().Set(a.reduction[0].Elem())
}

// Avg returns the average float value of the aggregate grouping
// panics if the field cannot be converted to an float64
func (a *AggregateResult) Avg(field string) float64 {
	sum := a.Sum(field)
	return sum / float64(len(a.reduction))
}

// Sum returns the sum value of the aggregate grouping
// panics if the field cannot be converted to an float64
func (a *AggregateResult) Sum(field string) float64 {
	var sum float64

	for i := range a.reduction {
		fVal, err := fieldValue(a.reduction[i].Elem(), field)
		if err != nil {
			panic(err)
		}

		sum += tryFloat(reflect.ValueOf(fVal))
	}

	return sum
}

// Count returns the number of records in the aggregate grouping
func (a *AggregateResult) Count() int {
	return len(a.reduction)
}

// FindAggregate returns an aggregate grouping for the passed in query
// groupBy is optional
func (s *Store) FindAggregate(dataType interface{}, query *Query, groupBy ...string) ([]*AggregateResult, error) {
	var result []*AggregateResult
	var err error
	err = s.Bolt().View(func(tx *bbolt.Tx) error {
		result, err = s.TxFindAggregate(tx, dataType, query, groupBy...)
		return err
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// TxFindAggregate is the same as FindAggregate, but you specify your own transaction
// groupBy is optional
func (s *Store) TxFindAggregate(tx *bbolt.Tx, dataType interface{}, query *Query,
	groupBy ...string) ([]*AggregateResult, error) {
	return s.aggregateQuery(tx, dataType, query, groupBy...)
}

func tryFloat(val reflect.Value) float64 {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		return float64(val.Int())
	case reflect.Uint, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uint8:
		return float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		return val.Float()
	default:
		panic(fmt.Sprintf("The field is of Kind %s and cannot be converted to a float64", val.Kind()))
	}
}
