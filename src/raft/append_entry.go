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

	// 加速日志恢复使用的几个字段
	Conflict bool
	XTerm    int
	XIndex   int
	XLen     int
}

// 发送心跳报或者写入日志
func (rf *Raft) appendEntries(heartbeat bool) {
	lastLog := rf.log.lastLog()
	for peer, _ := range rf.peers {
		if peer == rf.me {
			// 重置选举时间
			rf.resetElectionTimer()
			continue
		}
		// rules for leader 3
		if lastLog.Index >= rf.nextIndex[peer] || heartbeat {
			nextIndex := rf.nextIndex[peer]
			if nextIndex <= 0 {
				nextIndex = 1
			}
			if lastLog.Index+1 < nextIndex {
				nextIndex = lastLog.Index
			}
			prevLog := rf.log.at(nextIndex - 1)
			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				Entries:      make([]Entry, lastLog.Index-nextIndex+1),
				PrevLogIndex: prevLog.Index,
				PrevLogTerm:  prevLog.Term,
				LeaderCommit: rf.commitIndex,
			}
			// 写日志（2B）
			copy(args.Entries, rf.log.slice(nextIndex))
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
			//更新matchIndex和nextIndex
			match := args.PrevLogIndex + len(args.Entries)
			next := match + 1
			rf.nextIndex[serverID] = max(rf.nextIndex[serverID], next)
			rf.matchIndex[serverID] = max(rf.matchIndex[serverID], match)
			DPrintf("[%v]: %v append success next %v match %v", rf.me, serverID, rf.nextIndex[serverID], rf.matchIndex[serverID])
		} else if reply.Conflict {
			DPrintf("[%v]: Conflict from %v %#v", rf.me, serverID, reply)
			// 使用到日志压缩的参数字段
			// 如果Follower在对应位置没有Log，那么这里会返回 -1
			if reply.XTerm == -1 {
				rf.nextIndex[serverID] = reply.XLen
			} else {
				//如果Follower在对应位置的任期号不匹配，
				//它会拒绝Leader的AppendEntries消息，
				//并将自己的任期号放在XTerm中。
				lastLogInXTerm := rf.findLastLogInTerm(reply.XTerm)
				DPrintf("[%v]: lastLogInXTerm %v", rf.me, lastLogInXTerm)
				if lastLogInXTerm > 0 {
					rf.nextIndex[serverID] = lastLogInXTerm
				} else { // XIndex：这个是Follower中，对应任期号为XTerm的第一条Log条目的槽位号。
					rf.nextIndex[serverID] = reply.XIndex
				}
			}
			DPrintf("[%v]: leader nextIndex[%v] %v", rf.me, serverID, rf.nextIndex[serverID])
		} else if rf.nextIndex[serverID] > 1 {
			// 递减
			rf.nextIndex[serverID]--
		}
		rf.leaderCommitRule()
	}
}

// 倒查合适的term
func (rf *Raft) findLastLogInTerm(x int) int {
	for i := rf.log.lastLog().Index; i > 0; i-- {
		term := rf.log.at(i).Term
		if term == x {
			return i
		} else if term < x {
			break
		}
	}
	return -1
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
	for n := rf.commitIndex + 1; n <= rf.log.lastLog().Index; n++ {
		if rf.log.at(n).Term != rf.currentTerm {
			continue
		}
		counter := 1
		for serverID := 0; serverID < len(rf.peers); serverID++ {
			if serverID != rf.me && rf.matchIndex[serverID] >= n {
				counter++
			}
			if counter > len(rf.peers)/2 {
				rf.commitIndex = n
				DPrintf("[%v] leader尝试提交 index %v", rf.me, rf.commitIndex)
				rf.apply()
				break
			}
		}
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("[%d]: (term %d) follower 收到 [%v] AppendEntries %v, prevIndex %v, prevTerm %v", rf.me, rf.currentTerm, args.LeaderId, args.Entries, args.PrevLogIndex, args.PrevLogTerm)
	reply.Success = false
	reply.Term = rf.currentTerm
	// rules for servers all servers 2
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
	// append entries rpc 2
	// reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm.
	if rf.log.lastLog().Index < args.PrevLogIndex {
		reply.Conflict = true
		reply.XTerm = -1
		reply.XIndex = -1
		reply.XLen = rf.log.len()
		DPrintf("[%v]: Conflict XTerm %v, XIndex %v, XLen %v", rf.me, reply.XTerm, reply.XIndex, reply.XLen)
		return
	}
	if rf.log.at(args.PrevLogIndex).Term != args.PrevLogTerm {
		reply.Conflict = true
		xTerm := rf.log.at(args.PrevLogIndex).Term
		// 找到第一个不匹配的index
		for xIndex := args.PrevLogIndex; xIndex > 0; xIndex-- {
			if rf.log.at(xIndex-1).Term != xTerm {
				reply.XIndex = xIndex
				break
			}
		}
		reply.XTerm = xTerm
		reply.XLen = rf.log.len()
		DPrintf("[%v]: Conflict XTerm %v, XIndex %v, XLen %v", rf.me, reply.XTerm, reply.XIndex, reply.XLen)
		return
	}

	for idx, entry := range args.Entries {
		// append entries rpc 3
		// if an existing entry conflicts with a new one (same index but different terms),
		// delete the existing entry and all that follow it
		if entry.Index <= rf.log.lastLog().Index && rf.log.at(entry.Index).Term != entry.Term {
			rf.log.truncate(entry.Index)
			rf.persist()
		}
		// append entries rpc 4
		// append any new entries not already in the log
		if entry.Index > rf.log.lastLog().Index {
			rf.log.append(args.Entries[idx:]...)
			DPrintf("[%d]: follower append [%v]", rf.me, args.Entries[idx:])
			rf.persist()
			break
		}
	}

	// append entries rpc 5
	// if leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, rf.log.lastLog().Index)
		rf.apply()
	}
	reply.Success = true
}
