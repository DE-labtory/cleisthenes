<h2>PBFT Basic<h2/>


<h4>Distributed system error</h4> 

<ul>
<li> Fail-stop</li>
	
Although nodes crash each other, the failure of collision can be detected. And, the failure doesn't return an output of that failure. 
<li> Byzantine fault</li>
	
Much like to Fail-stop, nodes can collides each other, the failure is not easy to be detected, which can return wrong information or crashed output.</ul>

<h4>What is Byzantine fault?</h4>

<p align="center"><img src="../images/pbft_byzantine.png" width="600px" height="400px"></p>



<p>For example, if we assume that there are 3 nodes in whole network, there should be one node that play a role of commander of them. 
The Commander node provides a survey which can be voted by other 2 Lieutenant nodes, and after then the poll result will affect the decision 
making behavior. If these three nodes provide correct information to each other, they would reach correct consensus. However, there can exist any 
node that might be affected by outside effect so that providing information or making it hard to reach any consensus. (Byzantine node may not be exist, 
or it can exist more than 1 between them.) In this case, we call the node as Byzantine node. The Byzantine node causes failure to occur.</p>

<p>If there is one Byzantine node, the other one of 2 nodes would receive wrong information and correct information at the same time which
 leads to confusion. In this situation, there is no way to discern which information is correct, which makes failure to occur.

 By expanding example above, let's talk about PBFT.
</p>


<h4>PBFT concept</h4>

<p>PBFT is an agreement algorithm developed to ensure that all nodes participating in a distributed system can successfully reach an agreement when the distributed system is an asynchronous system in which a Byzantine node without a promised behavior exists. Existing BFT consensus algorithms agreed on issues that could only be agreed on in synchronous networks. But, with PBFT, nodes could be agreed on in asynchronous network with Byzantine nodes.</p>

<p>PBFT assumes that when there are 3*F+1 nodes in the whole network, number of F Byzantine fault can be defended, which means Byzantine fault doesn¡¯t occur and return correct output.
PBFT process commence when a client send a request to the primary node. This primary node collects all transaction to create a block. And then, through three steps below, it can reach consensus.</p>
<p align="center"><img src="../images/pbft_3steps.png" width="400px" height="200px"></p>



<h4>Pbft Main 3 STEPS</h4>

<ul>
<li> Pre prepare</li>
	
 The primary node that receiving request will send block to the other nodes in the whole network(sending pre prepare message), and all nodes will receive the block. 
Pre prepare message contains signature, meta data or sequence number that will help other nodes to verify its validity.
<li> Prepare</li>
	
 If all other nodes receive nodes, each node that have received block will send prepare a message which says the node received block to all the other nodes in the network.
 Every nodes will gather the result of the block verification of other nodes , and when the number of that validation is greater or equal to the number of 2/3 of whole nodes,
 the block is verified. (Prepare message informs that the nodes who sent message is valid at the moment.)	
<li> Commit</li>
	
After then, the verification result value will be projected to all other nodes. When the total of the block verification results is greater or equal to the number of 2/3 of whole nodes, the result value will be acknowledged as TRUE.

- If the verification result value is greater or equal to the number of 2/3 of whole nodes that the node is valid, the block will be committed to be added to the node's block chain.<br/>
   -However, if the verification result value is less than the number of 2/3 of whole nodes, the node is known as invalid node. So the block will not be add to its block chain.
    After these whole steps, the present status will be replied to the client.
   </ul>


