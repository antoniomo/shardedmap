# ShardedMap

Thread-safe concurrent maps for go. Inspired by
https://github.com/orcaman/concurrent-map, which is awesome, but doesn't let you
easily change the shards number upon creation, and doesn't provide ready-made
implementations for the most common keys, as it only supports `string` keys.
Also, this implementation should be more performant (benchmarks pending, but
using `runtime.memhash` is much faster than `fnv` hash, as proven by the
https://github.com/dgraph-io/ristretto team
[benchmarks](https://github.com/dgraph-io/ristretto/blob/master/z/rtutil_test.go)).

Here we provide ready-made implementation for `string`, `uint64`,
and `uuid`. Of course, `string` keys are the more common, but if you can use
`uint64` keys for your application, that could provide much better performance
in some cases. Also, `uuid`s are a common case of map keys. Instead of using
their `string` representation, here we provide ready-made `[16]byte` map key
support, which most Golang UUID libraries use as underlying type, so you can
store them without encoding/decoding, and with much less memory pressure.

Code generation for the exact value type has been considered, but for simplicity
(or just lazyness) I'm not doing that at the moment. If you need to squeeze that
extra performance, copy-paste and adapt the code to your usage.

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
	los, ok := strmap.LoadOrStore(a.ID, b)
	if !ok {
		panic("ARGH!")
	}

	fmt.Printf("%+v\n", a2)
	fmt.Printf("%+v\n", b2)
	fmt.Printf("%+v\n", los)

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
{ID:a V:1}
{ID:b V:2}
{ID:a V:1}
```

As expected.

## Inspiration

- https://github.com/orcaman/concurrent-map
- https://github.com/dgraph-io/ristretto
