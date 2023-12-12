## lab2A  
pass
![img.png](img.png)


## lab2B
### 选举增加条件
func RequestVote:
增加判定条件
需要Up To Date

func candidateRequestVote:
当主选完之后 有两个新增的参数
nextIndex
matchIndex

func leaderSendEntries
基于leader 的3.1判断是否日志写入冲突
这里如果冲突需要用到xterm等的 相关参数

func leaderCommitRule
需要参考leader rule 4

func apply
应用日志

func AppendEntries follower写日志
append entries rpc 2345

### 日志
写日志的时机：
1. 当主节点启动的时候需要写一条新日志更新index（心跳）
2.  append entries rpc 3 :修改或者删除日志
#### appendEntries
心跳或者满足leader 条件3， 则更新一波log的相关Index
主要点是args中的参数：
Entries
后两者主要用于日志的恢复
PrevLogIndex
PrevLogTerm
Raft有一个稍微复杂的选举限制（Election Restriction）。这个限制要求，在处理别节点发来的RequestVote RPC时，需要做一些检查才能投出赞成票。这里的限制是，节点只能向满足下面条件之一的候选人投出赞成票：
1. 候选人最后一条Log条目的任期号大于本地最后一条Log条目的任期号；
2. 或者，候选人最后一条Log条目的任期号等于本地最后一条Log条目的任期号，且候选人的Log记录长度大于等于本地Log记录的长度
#### 日志恢复
在AppendEntriesReply中增加如下字段，加速日志的恢复速度

```go
type AppendEntriesReply struct {
    // given args
    ...
    
    // 加速日志恢复使用的几个字段
    Conflict bool
    XTerm    int
    XIndex   int
    XLen     int
}
```
#### 日志应用到状态机
func applier()
#### 选主中增加参数
leaderElection：RequestVoteArgs
### 持久化
持久化的内容:Log、currentTerm、votedFor
实现函数persist和readPersist
持久化的时机: 
1. 节点启动的时候
2. 节点接收requestvote后如果投票选择是投，需要持久化