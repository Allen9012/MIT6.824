package raft

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
