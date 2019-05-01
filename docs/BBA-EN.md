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

In a BV-broadcast, each correct process `p` BV-broadcasts a binary value and obtains binary values. To store the values obtained by other processes, BV-broadcast provides `binValues` , which initialized to zero-set, which increases when new values are received. And BV-broadcast follows four properties:

* If at least `t + 1` correct processes BV-broadcast the same value `v`, `v` is added to the set `binValues` of each correct process `p`
* If `p` is correct and `v` is contained in `binValues` , then `v` has been BV-broadcast by a correct process.
* If a value `v` is added to the set `binValues` , of correct process `p`, eventually `v` which is contained in `binValues` is added to `binValues` at every correct processes.
* Eventually the set `binValues` of each correct process `p` is not empty

```go
func BVBroadcast(v binary) {
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
                    binValues.Union(bVal)
                }       	
            }
	}
    }()
}
```

* Suppose `v` is a value such that `t+1` correct processes invoke `BVBroadcast` , eventually `t+1` correct processes receive messages. And because one node exactly receive `t+1` messages from distinct processes (include myself), it broadcast `bVal` to all processes. As result `bVal received from (2t+1) diff processes` executed and `v` is added to `binValues` of all correct processes
* As there are at least `n-t` correct processes, each of them invokes `BVBroadcast`, `n-t >= 2t+1 = (t+1) + t`, and there's at least `t+1` correct processes, which enables adding value to `binValues`

 

## Randomized Byzantine consensus algorithm

```go
func propose(v binary) {
    estimated := v
    round := 0
    
    for {
        round++
        binValues := BinValues{}
        
        // result of this function is saved at binValues()
        BVBroadcast(Est(round, estimated)) 
        waitUntil(func() {
            return binValues.Size() != 0
        })
        
        for _, w := range binValues(r) {
            broadcast(Aux(round, w))
        }
        
        waitUntil(func() {
            // wait until set of (n - t) Aux(r, x) messages delivered from 
            // distinct processes where values is the set of values x
            // carried by these (n - t) messages
        })
        
        coin := random()
        if (len(values) == 1) {
            if (values[0] == coin && !done) {
                decide(values[0])
            }
            estimated = values[0]
        } else {
            estimated = coin
        }
    }
}
```

It requires t < n/3 to be tolerated. A process `p` invokes proposes `propose(v)`. `v` is one of 0 or 1. It decides its final value when it executes statement `decide(v)`.

* The local variable `estimated` of a process `p` keeps its current estimate of the decision. 
* The process proceed by consecutive asynchronous rounds and `BVBroadcast` call is associated with each round.
* `r` states the current round of process `p` 
*  `binValues`  is set of binary values used for `BVBroadcast` result



### Phase 1: Inform current estimate to others

```go
r++
// result of this function is saved at binValues
BVBroadcast(Est(round, est))
waitUntil(func() {
    return binValues.Size() != 0
})
```

* During a round `round` a process `p` first invokes `BVBroadcast(Est(round, est))` **to inform the other processes of the value of its current estimate `est` **
* Then `p` waits until its `binValues` no longer empty. Due to BV-Termination property, this eventually happens.
* When the predicate becomes satisfied, `binValues` contains at least one value: 0 or 1
* Due to the BV-Justification property, the values in `binValues` were `BVBroadcast` by correct processes



### Phase 2: Inform others' estimated values to others

```go
for _, w := range binValues {
	broadcast(Aux(round, w))
}

waitUntil(func() {
    // wait until set of (n - t) Aux(round, x) messages delivered from 
    // distinct processes where values is the set of values x
    // carried by these (n - t) messages
})
```

* Each correct process `p` invokes `broadcast(Aux(round, w))` where `w` is a value that belongs to `binValues` .
* Byzantine process can broadcast an arbitrary binary value. So this phase is for **informing other processes of estimated values of estimate values that have been `BVBroadcast` by correct processes**
* Wait until `n - t` messages delivered from distinct processes, for discarding values sent only by Byzantine processes. And value sent by `n - t` messages saved to `values`.
* `values` contains only the values originated from correct processes. In other words, the set `values` cannot contain value broadcast only by Byzantine processes.
* If both `p_i` and `p_j` are correct processes, and if `values_i = {v}`  and `values_j = {w}` for round `r` , then `v = w`
  * Because `p_i` has `values_i = {v}` , `p_i` has received the same message `Aux(round, v)` from at least `n - t` different processes.
  * Because at most `t` processes can be Byzantine, it follows that `p_i` received `Aux(round, v)` from at least `n - 2t` different correct processes.
  * Because `n - 2t >= t + 1` , `p_i` received at least from `t + 1` correct processes
  * And another correct process `p_j` has `values_j = {w}` , which received from `Aux(round, w)` from at least `n - t` different processes
  * Because `(n - t) + (t + 1) > n` , at least one correct process, let's say `p_x` , sent `Aux(round, v)` to `p_i` and `Aux(round, w)` to `p_j`.
  * Since `p_x` is correct, it sent the same message to all processes. Therefore `v = w`



### Phase 3: Decide or not

```go
coin := random()
if (len(values) == 1) {
	if (values[0] == s && !done) {
		decide(values[0])
	}
	estimated = values[0]
} else {
	estimated = coin
}
```

* `coin` is network global entity that delivers the same sequence of random bits `b(1), b(2), ..., b(r)` to processes and each `b(r)` has the value 0 or 1 with probability 1/2.
  * In addition to being random, this bit has to following global property. the `r` th invocation of `random()` by a correct process `p`, returns it the bit `b(r)` , it means that the r times call of `random()` makes return value `coin` to all of correct processes.
  * `coin` , a common coin is built in such a way that the correct processes need to cooperate to compute the value of each bit `b(r)`.
* Correct process `p_i` obtains the common coin value `coin` associated with the current round
  * In the case of `len(values) == 2` , it means that both the value 0 and 1 are estimate values of correct processes, then `p` adopts the value `coin` of the common coin
  * In the case of `len(values) == 1`  and `coin == v`, then `decide(values[0])` Otherwise it adopts `values[0]` as its new estimate
* function `decide` allows `p` to decide but does not stop its execution. A process executes round forever.

* A decided value is a value proposed by a correct process
* No two correct processes decide different values



## Conclusion

This Binary Byzantine Agreement algorithm suited to asynchronous systems composed of `n` processes, and where up to `t < n/3` processes may have a Byzantine behavior. This algorithm use Rabin's common coin and BV-broadcast algorithm, both of which helps to guarantee a value broadcast only by Byzantine processes is never agreeed to the correct processes.