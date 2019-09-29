# ShardedMap

Thread-safe concurrent maps for go. Initially inspired by
https://github.com/orcaman/concurrent-map, which is awesome, but doesn't let you
easily change the shards number upon creation, and doesn't provide ready-made
implementations for the most common keys, as it only supports `string` keys.
Later heavily influenced by https://github.com/tidwall/shardmap.

This implementation should be more performant than Orcaman's in most cases, due
to the customizable number of shards, better default, and use of a faster hash.

Here we provide ready-made implementation for `string`, `uint64`,
and `uuid`. Of course, `string` keys are the more common, but if you can use
`uint64` keys for your application, that could provide much better performance
in some cases.

Also, `uuid`s are a common case of map keys. Instead of using their `string`
representation, here we provide ready-made `[16]byte` map key support, which
most Golang UUID libraries use as underlying type, so you can store them without
encoding/decoding, and with much less memory pressure.

Code generation for the exact value type has been considered, but for simplicity
(or just lazyness) I'm not doing that at the moment. If you need to squeeze that
extra performance, copy-paste and adapt the code to your usage. Also PRs and
motivational issues are welcome!

## Which concurrent map to use

That very much depends on your concurrency needs, your hardware, and your
requirements, but a few rules of thumb that might or might not help:

- If you have less than 4 cores, a single map with an RWMutex is all you need.
- Many cores and goroutines:
  - Tons of reads, not many writes: `sync.Map` is appropriate.
  - Mixed reads and writes, go for a sharded map implementation. At the moment
    [\[1\]](https://github.com/orcaman/concurrent-map) is a veteran library,
    well tested. [\[2\]](https://github.com/tidwall/shardmap) is a newcomer but
    looks very good. This library is inspired on both. It should be a bit more
    performant than [\[1\]](https://github.com/orcaman/concurrent-map), while
    using standard Go maps and slightly less dependencies than
    [\[2\]](https://github.com/tidwall/shardmap). Also, this library has native
    support for `uint64` and `UUID` keys if you want that. However due to my
    lazyness and because I just use this for personal projects, this lib is not
    so well tested. Caveat Emptor (and PRs are welcome, or file an issue to
    motivate me into writing those tests!).
- Other invariants to keep together with the map semantics: Write your own data
  structure or pick one implementation you like and fork it, then add those
  invariants there.
- Mixed value types, TTL, queries... You don't want a map, you want either a
  cache or an in-memory database, depending on your exact requirements. Check
  for example [\[3\]](https://github.com/dgraph-io/ristretto) and
  [\[4\]](https://github.com/dgraph-io/badger).

## Installation

`go get -u github.com/antoniomo/shardedmap`

## Sample usage

```go
package main

import (
	"fmt"

	"github.com/antoniomo/shardedmap"
)

type Data struct {
	ID string
	V  int
}

func main() {
	strmap := shardedmap.NewStrMap(64)
	a := Data{ID: "a", V: 1}
	b := Data{ID: "b", V: 2}

	strmap.Store(a.ID, a)
	strmap.Store(b.ID, b)

	a2, ok := strmap.Load(a.ID)
	if !ok {
		panic("ARGH!")
	}
	b2, ok := strmap.Load(b.ID)
	if !ok {
		panic("ARGH!")
	}

	fmt.Println("Same with range (note the random order):")
	strmap.Range(func(key string, value interface{}) bool {
		fmt.Printf("Key: %s, Value: %+v\n", key, value)
		return true
	})

	fmt.Println("LoadOrStore over a.ID")
	los, ok := strmap.LoadOrStore(a.ID, b)
	if !ok {
		panic("ARGH!")
	}
	fmt.Printf("Key: %s, Value: %+v\n", a.ID, los)

	strmap.Delete(a.ID)
	strmap.Delete(b.ID)
	_, ok = strmap.Load(a.ID)
	if ok {
		panic("ARGH!")
	}
	_, ok = strmap.Load(b.ID)
	if ok {
		panic("ARGH!")
	}
}

```

The output of that should be:

```
Key: a, Value: {ID:a V:1}
Key: b, Value: {ID:b V:2}
Same with range (note the random order):
Key: b, Value: {ID:b V:2}
Key: a, Value: {ID:a V:1}
LoadOrStore over a.ID
Key: a, Value: {ID:a V:1}
```

As expected.

## References

- [\[1\]](https://github.com/orcaman/concurrent-map) https://github.com/orcaman/concurrent-map
- [\[2\]](https://github.com/tidwall/shardmap) https://github.com/tidwall/shardmap
- [\[3\]](https://github.com/dgraph-io/ristretto) https://github.com/dgraph-io/ristretto
- [\[4\]](https://github.com/dgraph-io/badger) https://github.com/dgraph-io/badger
