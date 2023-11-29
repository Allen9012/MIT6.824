package raft

/**
  Copyright © 2023 github.com/Allen9012 All rights reserved.
  @author: Allen
  @since: 2023/11/28
  @desc:
  @modified by:
**/

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Entry
	LeaderCommit int
}

type AppendEntriesReply struct {
	// given args
	Term    int
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm

	//
	Conflict bool
	XTerm    int
	XIndex   int
	XLen     int
}

// 发送心跳报或者写入日志
func (rf *Raft) appendEntries(heartbeat bool) {
	for peer, _ := range rf.peers {
		if peer == rf.me {
			// 重置选举时间
			rf.resetElectionTimer()
			continue
		}
		// TODO 日志
		//rules for leader 3
		if heartbeat {
			args := AppendEntriesArgs{
				Term:     rf.currentTerm,
				LeaderId: rf.me,
				Entries:  make([]Entry, 0),
			}
			// 写日志（2B）
			go rf.leaderSendEntries(peer, &args)
		}
	}
}

func (rf *Raft) leaderSendEntries(serverID int, args *AppendEntriesArgs) {
	var reply AppendEntriesReply
	ok := rf.sendAppendEntries(serverID, args, &reply)
	if !ok {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 更新term
	if reply.Term > rf.currentTerm {
		rf.setNewTerm(reply.Term)
		return
	}

	if args.Term == rf.currentTerm {
		// rules for leader 3.1
		if reply.Success {

		} else if reply.Conflict {

			//} else if rf.nextIndex[serverID] > 1 {

		}
		rf.leaderCommitRule()
	}
}

func (rf *Raft) sendAppendEntries(serverID int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[serverID].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) leaderCommitRule() {
	// leader rule 4
	if rf.state != Leader {
		return
	}
	// TODO 写日志使用
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("[%d]: (term %d) follower 收到 [%v] AppendEntries %v, prevIndex %v, prevTerm %v", rf.me, rf.currentTerm, args.LeaderId, args.Entries, args.PrevLogIndex, args.PrevLogTerm)
	// rules for servers all servers 2
	//reply.Success = false
	reply.Term = rf.currentTerm
	if args.Term > rf.currentTerm {
		rf.setNewTerm(args.Term)
		return
	}
	// append entries rpc 1
	if args.Term < rf.currentTerm {
		return
	}
	// 满足leader条件清空election倒计时
	rf.resetElectionTimer()

	// candidate rule 3
	if rf.state == Candidate {
		rf.state = Follower
	}
	// TODO 日志
	// append entries rpc 2
	// append entries rpc 3
	// append entries rpc 4
	// append entries rpc 5
	reply.Success = true
}
