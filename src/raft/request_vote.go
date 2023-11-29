package raft

import "sync"

/**
  Copyright © 2023 github.com/Allen9012 All rights reserved.
  @author: Allen
  @since: 2023/11/28
  @desc:
  @modified by:
**/

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // currentTer, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

// example RequestVote RPC handler. follower handler
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// rule for servers all servers 2
	if args.Term > rf.currentTerm { // Term合法就更新raft 当前term状态
		rf.setNewTerm(args.Term)
	}

	// request vote rpc receiver 1
	if args.Term < rf.currentTerm { // term 不正确
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
	}

	// request vote rpc receiver 2
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId { // 没有投票或者已经投票过指定
		rf.votedFor = args.CandidateId //投给 candidate
		reply.VoteGranted = true       // 表示投candidate
		rf.resetElectionTimer()
	} else {
		reply.VoteGranted = false
	}
	// 最后更新term（统一reply对应当前机器raft的term）
	reply.Term = rf.currentTerm
}

// candidate candidateRequestVote
func (rf *Raft) candidateRequestVote(serverId int, args *RequestVoteArgs, voteCounter *int, becomeLeader *sync.Once) {
	DPrintf("[%d]: term %v send vote request to %d\n", rf.me, args.Term, serverId)
	reply := RequestVoteReply{}
	// 对指定server发送rpc 请求投票
	ok := rf.sendRequestVote(serverId, args, &reply)
	if !ok {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 收到reply后处理
	if reply.Term > args.Term {
		DPrintf("[%d]: %d 在新的term，更新term，结束\n", rf.me, serverId)
		rf.setNewTerm(reply.Term)
		return
	}
	// request vote rule 1
	if reply.Term < args.Term {
		DPrintf("[%d]: %d 的term %d 已经失效，结束\n", rf.me, serverId, reply.Term)
		return
	}
	if !reply.VoteGranted { // 请求投票中发现不投票
		DPrintf("[%d]: %d 没有投给me，结束\n", rf.me, serverId)
		return
	}
	DPrintf("[%d]: from %d term一致，且投给%d\n", rf.me, serverId, rf.me)
	*voteCounter++

	// 计数满足条件, 且再次判断term
	if *voteCounter > len(rf.peers)/2 &&
		rf.currentTerm == args.Term &&
		rf.state == Candidate {
		// once 保证只会切换和发送heartbeat一次
		becomeLeader.Do(func() {
			DPrintf("[%d]: 获得多数选票，可以提前结束\n", rf.me)
			rf.state = Leader
			// TODO 日志

			DPrintf("[%d]: leader - nextIndex %#v", rf.me, rf.nextIndex)
			// 立刻发送一次心跳
			rf.appendEntries(true)
		})
	}
}
