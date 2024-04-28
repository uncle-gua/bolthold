// Copyright 2019 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package bolthold_test

import (
	"fmt"
	"testing"

	"github.com/uncle-gua/bolthold"
	"go.etcd.io/bbolt"
)

func TestForEach(t *testing.T) {
	testWrap(t, func(store *bolthold.Store, t *testing.T) {
		insertTestData(t, store)
		for _, tst := range testResults {
			t.Run(tst.name, func(t *testing.T) {
				count := 0
				err := store.ForEach(tst.query, func(record *ItemTest) error {
					count++

					found := false
					for i := range tst.result {
						if record.equal(&testData[tst.result[i]]) {
							found = true
							break
						}
					}

					if !found {
						if testing.Verbose() {
							return fmt.Errorf("%v was not found in the result set! Full results: %v",
								record, tst.result)
						}
						return fmt.Errorf("%v was not found in the result set!", record)
					}

					return nil
				})
				if count != len(tst.result) {
					t.Fatalf("ForEach count is %d wanted %d.", count, len(tst.result))
				}
				if err != nil {
					t.Fatalf("Error during ForEach iteration: %s", err)
				}
			})
		}
	})
}

func TestForEachInBucket(t *testing.T) {
	testWrapWithBucket(t, func(store *bolthold.Store, bucket *bbolt.Bucket, t *testing.T) {
		insertBucketTestData(t, store, bucket)
		for _, tst := range testResults {
			t.Run(tst.name, func(t *testing.T) {
				count := 0
				err := store.ForEachInBucket(bucket, tst.query, func(record *ItemTest) error {
					count++

					found := false
					for i := range tst.result {
						if record.equal(&testData[tst.result[i]]) {
							found = true
							break
						}
					}

					if !found {
						if testing.Verbose() {
							return fmt.Errorf("%v was not found in the result set! Full results: %v",
								record, tst.result)
						}
						return fmt.Errorf("%v was not found in the result set!", record)
					}

					return nil
				})
				if count != len(tst.result) {
					t.Fatalf("ForEach count is %d wanted %d.", count, len(tst.result))
				}
				if err != nil {
					t.Fatalf("Error during ForEach iteration: %s", err)
				}
			})
		}
	})
}

func TestForEachKeyStructTag(t *testing.T) {
	testWrap(t, func(store *bolthold.Store, t *testing.T) {
		type KeyTest struct {
			Key   int `boltholdKey:"Key"`
			Value string
		}

		key := 3

		err := store.Insert(key, &KeyTest{
			Value: "test value",
		})

		if err != nil {
			t.Fatalf("Error inserting KeyTest struct for Key struct tag testing. Error: %s", err)
		}

		err = store.ForEach(bolthold.Where(bolthold.Key).Eq(key), func(result *KeyTest) error {
			if result.Key != key {
				t.Fatalf("Key struct tag was not set correctly.  Expected %d, got %d", key, result.Key)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Error running ForEach in TestKeyStructTag. ERROR: %s", err)
		}
	})
}

func TestForEachKeyStructTagIntoPtr(t *testing.T) {
	testWrap(t, func(store *bolthold.Store, t *testing.T) {
		type KeyTest struct {
			Key   *int `boltholdKey:"Key"`
			Value string
		}

		key := 3

		err := store.Insert(&key, &KeyTest{
			Value: "test value",
		})

		if err != nil {
			t.Fatalf("Error inserting KeyTest struct for Key struct tag testing. Error: %s", err)
		}

		err = store.ForEach(bolthold.Where(bolthold.Key).Eq(key), func(result *KeyTest) error {
			if result.Key == nil || *result.Key != key {
				t.Fatalf("Key struct tag was not set correctly.  Expected %d, got %d", key, result.Key)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Error running ForEach in TestKeyStructTag. ERROR: %s", err)
		}
	})
}
