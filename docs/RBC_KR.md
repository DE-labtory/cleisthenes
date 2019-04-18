# Reliable Broadcast



### overview

구현에 따라 다르지만 일반적인 브로드캐스트는 메시지 전달을 보장하지 않고 sender가 crash 되는 상황(메시지를 모든 노드에게 보내지 못한 상황)을 고려하지 않는다. 하지만 RBC는 메시지 전달을 보장하고 sender가 crash되는 상황도 부분적으로 해결한다. 그리고 이 논문에서 소개하는 Atomic Broadcast는 sender가 crash되는 상황을 보장하고 메시지의 순서까지 보장하는 Broadcast이며 소개될 RBC는 Atomic Broadcast이다.



논문은 ABC(Atomic Broadcast)를  `threshold encryption` 을 사용하여 효율성을 올린 ACS(Asynchronous Common Subset)으로 나타내었고 ACS는 RBC(Reliable Broadcast)와 BBA(Binary Byzantine Agreement) 컴포넌트로 이루어져 있다. 이 중 RBC는 Reliable Broadcast를 하고 BBA는 consensus와 Atomic한 message의 ordering을 담당한다.

HBBFT는 block 전체를 보내는 것이 아닌 block을 네트워크의 노드 수만큼 쪼개서 조각들을 보낸다. 이를 위해 RBC에서는 Erasure coding이라는 기법을 사용한다. 

### Erasure coding   

Erasure coding은 FEC(Forward Error Correction) 방식 중 하나이며 ` 에러 검출` 과 `에러 정정 ` 을 할 수 있다. RBC에서는 Erasure coding 기법 중 Reed-solomon 기법을 사용한다. 이는 원본 데이터를 `n개`로 쪼갠 후 `k개의 parity`를 추가하여 최대 k개의 데이터가 손실 되어도 n개의 데이터만 살아 있으면 원본 데이터 복구가 가능한 기법이다. 예제와 자세한 설명은 [여기](<https://www.backblaze.com/blog/reed-solomon/>)에 자세히 설명되어있다.



### RBC Algorithm

![image-20190417230329518](/Users/kim/DE-labtory/RBC.png)

네트워크 상에서 합의 할 내용물을 Block이라고 가정한다.

- 네트워크 상에는 `N`개의 노드가 존재한다.
- Block을 `N`개의 조각으로 쪼갠다. 그리고 `N-2f`개의 parity를 생성하는 erasure coding을 사용한다.
- `s(j)`는 쪼개진 조각이고 전달 할 메시지는 `VAL(h, b(j), s(j))` 로 구성된다. `h` : `s(j)`의 Merkle tree root, `b(j)` : `s(j)`가 이루는 Merkle tree branch

1. `P(sender)`에게 `VAL(h, b(j), s(j))`을 받으면 자신과 `Sender`를 제외한 노드에게 `ECHO(h, b(j), s(j))`를  multicast한다.
2. `P(j)`로 부터 `ECHO(h, b(j), s(j))`를 받으면 merkle정보인 `s(j)`와 `b(j)`를 이용해 `h`가 유효한지 확인한다. 유효하지 않으면 버린다.
3. `N-f`의 노드로 부터 valid한 `ECHO`를 받을 때,
   - `N-2f`개의 leave로부터  `s(j)`를 interpolate한다.
   - interpolate된 merkle root를 확인한다. 유효하지 않으면 버린다.
   - `READY(h)`를 아직 보내지 않았다면 `READY(h)`를 multicast한다.
4. `READY(h)` 메시지와 일치하는 `f + 1`개의 메시지를 받았고 `READY`를 아직 보내지 않았다면 `READY`를 multicast한다.
5. `READY(h)` 메시지와 일치하는 `2f + 1`개의 메시지를 받았을 때, `N-2f`개의 `ECHO` 메시지를 기다린 다음  `VALUE`를 Decoding한다.

> `READY(h)`는 `N-f`개의 노드로 부터 `ECHO`메시지를 받았을 때의 상태이다.

그리고 RBC는 다음을 만족해야 한다.

- Agreement : 정직한 두 노드가 VALUE v와 v`를 전달하면 v=v1이다.
- Totality : 어떤 정직한 노드가 v를 전달하면 모든 정직한 노드는 v를 전달한다.
- Validity : sender가 정직하고 input이 v라면 모든 정직한 노드는 v를 전달한다.