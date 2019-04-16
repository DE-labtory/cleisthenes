# Binary Byzantine Agreement

## Overview

Binary Byzantine Agreement tries to solve consensus problem in asynchronous distributed systems where processes or node can commit Byzantine failures. A process can have a Byzantine behavior when it arbitrary deviates from its intended behavior, it then commits a Byzantine failure.

This bad behavior can be intentional or simply the result of temporary fault that altered the its normal behavior in unpredictable way. One example of bad behavior is crash of process.



## Binary Byzantine consensus problem

The Binary Byzantine consensus problem is that each correct process `p(i)` proposes a value `v(i)` (0 or 1). And each of members in network has to decide a value such that the following properties are satisfied:

* A decided value was proposed by a correct process
* No two correct processes decide different values
* A correct process decides at most once.
* Each correct process decides, and terminate consensus

These properties states that value proposed only by faulty node can be decided. In other words, if all correct processes propose the same value `v`, the value `v'` cannot be decided. 



## Requirements

The requirements for this algorithm to work is if `n` is the number of members that engaged in the agreement, **the number of failed member `t` must satisfies `n > 3t` condition.**



## Binary-Value Broadcast

In a BV-broadcast, each correct process `p(i)` BV-broadcasts a binary value and obtains binary values. To store the values obtained by other processes, BV-broadcast provides `binValues(i)` , which initialized to zero-set, which increases when new values are received. And BV-broadcast follows four properties:

* If at least `t + 1` correct processes BV-broadcast the same value `v`, `v` is added to the set `binValues(i)` of each correct process `p(i)`
* If `p(i)` is correct and `v` is contained in `binValues(i)` , then `v` has been BV-broadcast by a correct process.
* If a value `v` is added to the set `binValues(i)` , of correct process `p(i)`, eventually `v` which is contained in `binValues(i)` is added to `binValues(j)` at every correct process `p(j)`
* Eventually the set `binValues(i)` of each correct process `p(i)` is not empty

```go
func BVBroadcast(v Value) {
    broadcast(BVal(v))
    go func() {
        for {
            select {
            case bVal := <- bValChan:
                if bVal received from (t+1) diff processes && !isBroadcast(bVal) {
                    broadcast(bVal)
                    continue;
                }

                if bVal received from (2t+1) diff processes {
                    binValues(i) = append(binValues(i), bVal)
                }       	
            }
	}
    }()
}
```

* Suppose `v` is a value such that `t+1` correct processes invoke `BVBroadcast` , eventually `t+1` correct processes receive messages. And because one node exactly receive `t+1` messages from distinct processes (include myself), it broadcast `bVal` to all processes. As result `bVal received from (2t+1) diff processes` executed and `v` is added to `binValues(i)` of all correct processes
* As there are at least `n-t` correct processes, each of them invokes `BVBroadcast`, `n-t >= 2t+1 = (t+1) + t`, and there's at least `t+1` correct processes, which enables adding value to `binValues`

 

## Randomized Byzantine consensus algorithm

```go
func propose(i string, v binary) {
    est := v
    r := 0
    
    for {
        r++
        binValues := BinValues{}
        
        // result of this function is saved at binValues(r(i))
        BVBroadcast(Est(r(i), est)) 
        waitUntil(func() {
            return len(binValues(r(i))) != 0
        })
        
        for _, w := range binValues(r(i)) {
            broadcast(Aux(r(i), w))
        }
        
        waitUntil(func() {
            // wait until set of (n - t) Aux(r, x) messages delivered from 
            // distinct processes where values(i) is the set of values x
            // carried by these (n - t) messages
        })
        
        s := random()
        if (len(values(i)) == 1) {
            if (values(i)[0] == s && !done) {
                decide(i, values(i)[0])
            }
            est = values(i)[0]
        } else {
            est = s
        }
    }
}
```

It requires t < n/3 to be tolerated. A process `p(i)` invokes proposes `propose(i, v(i))`. `v(i)` is one of 0 or 1. It decides its final value when it executes statement `decide(i, v)`.

* The local variable `est` of a process `p(i)` keeps its current estimate of the decision. 
* The process proceed by consecutive asynchronous rounds and `BVBroadcast` call is associated with each round.
* `r` states the current round of process `p(i)` 
*  `binValues(r(i))`  is set of binary values used for `BVBroadcast` result



### Phase 1: Inform current estimate to others

```go
r++
// result of this function is saved at binValues(r(i))
BVBroadcast(Est(r(i), est))
waitUntil(func() {
	return len(binValues(r(i))) != 0
})
```

* During a round `r(i)` a process `p(i)` first invokes `BVBroadcast(Est(r(i), est))` **to inform the other processes of the value of its current estimate `est` **
* Then `p(i)` waits until its `binValues` no longer empty. Due to BV-Termination property, this eventually happens.
* When the predicate becomes satisfied, `binValues` contains at least one value: 0 or 1
* Due to the BV-Justification property, the values in `binValues` were `BVBroadcast` by correct processes



### Phase 2: Inform others' estimated values to others

```go
for _, w := range binValues(r(i)) {
	broadcast(Aux(r(i), w))
}

waitUntil(func() {
    // wait until set of (n - t) Aux(r, x) messages delivered from 
    // distinct processes where values(i) is the set of values x
    // carried by these (n - t) messages
})
```

* Each correct process `p(i)` invokes `broadcast(Aux(r(i), w))` where `w` is a value that belongs to `binValues` .
* Byzantine process can broadcast an arbitrary binary value. So this phase is for **informing other processes of estimated values of estimate values that have been `BVBroadcast` by correct processes**
* Wait until `n - t` messages delivered from distinct processes, for discarding values sent only by Byzantine processes. And value sent by `n - t` messages saved to `values`.
* `values` contains only the values originated from correct processes. In other words, the set `values` cannot contain value broadcast only by Byzantine processes.
* If both `p(i)` and `p(j)` are correct processes, and if `values(i) = {v}`  and `values(j) = {w}` for round `r` , then `v = w`
  * Because `p(i)` has `values(i) = {v}` , `p(i)` has received the same message `Aux(r(i), v)` from at least `n - t` different processes.
  * Because at most `t` processes can be Byzantine, it follows that `p(i)` received `Aux(r(i), v)` from at least `n - 2t` different correct processes.
  * Because `n - 2t >= t + 1` , `p(i)` received at least from `t + 1` correct processes
  * And another correct process `p(j)` has `values(j) = {w}` , which received from `Aux(r(j), w)` from at least `n - t` different processes
  * Because `(n - t) + (t + 1) > n` , at least one correct process, let's say `p(x)` , sent `Aux(r(i), v)` to `p(i)` and `Aux(r(j), w)` to `p(j)`.
  * Since `p(x)` is correct, it sent the same message to all processes. Therefore `v = w`



### Phase 3: Decide or not

```go
s := random()
if (len(values(i)) == 1) {
	if (values(i)[0] == s && !done) {
		decide(i, values(i)[0])
	}
	est = values(i)[0]
} else {
	est = s
}
```

* `s` is network global entity that delivers the same sequence of random bits `b(1), b(2), ..., b(r)` to processes and each `b(r)` has the value 0 or 1 with probability 1/2.
  * In addition to being random, this bit has to following global property. the `r` th invocation of `random()` by a correct process `p(i)`, returns it the bit `b(r)` , it means that the r times call of `random()` makes return value `s` to all of correct processes.
  * `s` , a common coin is built in such a way that the correct processes need to cooperate to compute the value of each bit `b(r)`.
* Correct process `p(i)` obtains the common coin value `s` associated with the current round
  * In the case of `len(values(i)) == 2` , it means that both the value 0 and 1 are estimate values of correct processes, then `p(i)` adopts the value `s` of the common coin
  * In the case of `len(values(i)) == 1`  and `s == v`, then `decide(i, values(i)[0])` Otherwise it adopts `values(i)[0]` as its new estimate
* function `decide` allows `p(i)` to decide but does not stop its execution. A process executes round forever.

* A decided value is a value proposed by a correct process
* No two correct processes decide different values



## Conclusion

This Binary Byzantine Agreement algorithm suited to asynchronous systems composed of `n` processes, and where up to `t < n/3` processes may have a Byzantine behavior. This algorithm use Rabin's common coin and BV-broadcast algorithm, both of which helps to guarantee a value broadcast only by Byzantine processes is never agreeed to the correct processes.