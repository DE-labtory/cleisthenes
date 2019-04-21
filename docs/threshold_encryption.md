# Threshold Encryption
*Threshold encryption* is one of cryptosystem, if in order to decrypt an encrypted message or to sign a message, several 
parties (more than some threshold number) must cooperate in the decryption or signature protocol.
The message is encrypted using a `public key` and the set of private key is distributed to the participating parties.
Only by possessing more than a threshold number of the secret, node can decrypt original message.

### Idea
The idea of threshold cryptography is to protect information (or computation) by fault-tolerance distributing it among a
cluster of cooperating computers. First consider the fundamental problem of threshold cryptography, a problem of secure
sharing of a secret. A secret sharing scheme allows one to distribute a piece of secret information to several participants
in a way that meets the following requirements.
1. No group of corrupt servers (smaller than a given threshold) can figure out what the secret is, even if they cooperate.
2. When it becomes necessary that the secret information be reconstructed, a large enough number of servers (a number
larger than the above threshold) can always do it.
  
### Goal
Threshold encryption implements the most secure cryptosystems and signature schemes.
The following are some of the various considerations threshold encryption makes when modeling computer fault.
* The size of the threshold : What fraction of the servers can be corrupted by the adversary without any harm to the service
that these servers implement?
* Efficiency considerations : How much communication, storage, and computation do these fault-tolerant protocols require?
* Model of communication : How realistic are the requirements we place on it? Do we require synchronous or partially synchronous
communication, authenticated broadcast and secure links between servers?
* Type of adversary : How does the adversary choose which players to corrupt? Can a server securely erase its local data so
that it cannot be retrieved by the adversary once the server is infiltrated?

### In HoneybadgerBFT
In HBBFT, threshold encryption is used when node encrypts and sends its message.
Node picks `B / N` (B is the batch size, and N is the number of elements in buffer) elements to its queue, encrypts them using public key which is made using several private keys.
When decrypting message, nodes share their decrypted digest. It leads network more robust.
Threshold encryption in HBBFT works as follow.

* `TPKE.SetUp` : Make one master public key, and several private keys using public key.
* `TPKE.Encrypt` : Encrypt message using master public key.
* `TPKE.DecShare` : Node who want to decrypt original message have to collect more than threshold share of decryption.
* `TPKE.Decrypt` : Decrypt message based on Shamir's Secret Sharing.

### Reference
<a href="http://wiki.c2.com/?ThresholdCryptography">Threshold Cryptography</a>
<a href="http://groups.csail.mit.edu/cis/cis-threshold.html">Cryptography and Information Security Group Research Project: Threshold Cryptology</a>