### GOSSIP PROTOCOL
A Gossip protocol is a procedure or process of *peer-to-peer communication* that is based on the way that 
**epidemic spread.**<br>
Because of it's epidemic character, it is often called **Epidemic protocol.**<br>
Some distributed systems use gossip protocol to ensure that data is routed to all members of and ad-hoc network.<br>
Some ad-hoc networks have no central registry and the only way to spread common data is to rely on each member to pass it
along to their neighbors.

#### Gossip
The concept of gossip communication can be illustrated by the analogy of spreading rumors.<br>
For example, there are several pair of people, they are sharing a gossip.<br>
A person who first heard gossip say her neighborhood, she say it to other neighborhood.<br>
As a result, everyone knows the gossip quickly.<br>
This is the principle of **Gossip Protocol**.<br>

In the real network, gossip protocol works as follows.<br> 
First, select the random node and send message periodically.<br>
Next, the node who got the message will send it to other random which is selected at random.<br>
This is called **Push gossip**.

But, Push gossip is inefficient when message is spread in almost every node.<br>
In this case, **Pull gossip** that asks if there is new gossip message is more efficient.<br> 

#### Where Gossip Protocol is used?
Gossip Protocol is used in **distributed system**.<br>
In distributed network system, the number of nodes increase rapidly, so abnormal opreation or packet drop also occur 
frequently.<br>
Therefore multi-cast is inadequate because it doesn't have fault-tolerance and scalability.<br>
But Gossip protocol is light-weight in large group of network.<br>
Message spreads very quickly, and network can have strong fault-tolerance.<br>
*Hyperledger fabric* and *DevP2P* in etherium core also use gossip protocol because of these characters.

#### Topology-Aware Gossip
If router selects uninfected node in uniformly random in process of spreading gossip message, router may have load of
`O(N)`.<br>
When `n(i)` nodes exist in subnet, in case of choosing a uninfected node with a probability of `1 - (1 / n(i))`, router 
only have load of `O(1)`. 