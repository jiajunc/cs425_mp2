package main

import (
	// "bytes"
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"./mp2util"
)

/*
	Helper function for introducer. When received a "join" msg(status), reply the membership list.
*/
func GiveMemberList(pc net.PacketConn, addr net.Addr, memberList *[]MemberID) {
	reply, _ := json.Marshal(*memberList)
	pc.WriteTo(reply, addr)
}

/*
	This function will run forever in go-routine.
	When receive a message, it will be added to msgs-queue, which will be handled by Pinging().
	Each messgae will be recorded in log according to each type.
*/
func UDPListening(memberList *[]MemberID, msgQueue *[]GossipMessage) {
	pc, err := net.ListenPacket("udp", mp2util.GetLocalIP().String()+":12345")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	for {
		buf := make([]byte, 1024)
		n, addr, err := pc.ReadFrom(buf)

		if err != nil {
			continue
		}
		// Offer an ack when it is a normal ping.
		if buf[0] == 110 && buf[1] == 117 && buf[2] == 108 && buf[3] == 108 {
			pc.WriteTo(buf[:n], addr)
			continue
		}

		b := buf[:n]
		var message GossipMessage
		err2 := json.Unmarshal(b, &message)

		if err2 != nil {
			log.Println("Unmarshal fails: ", err2)
			continue
		}

		addNewMsg := true
		// Check the message type.
		switch message.Status {
		case 1: // New join.
			if !MemberInList(message.Member, memberList) {
				*memberList = append(*memberList, message.Member)

				logMsg, _ := json.Marshal(message.Member)
				mp2util.WriteLog("Add: New joining member to local memberList:" + string(logMsg))

				if mp2util.GetLocalIP().String() == "172.22.158.188" {
					GiveMemberList(pc, addr, memberList)
					addNewMsg = false
				}
				fmt.Println("New member joined the group:")
				fmt.Println(message.Member)
				fmt.Println()
			}
		case 2: // Voluntarily leave.
			for i, Member := range *memberList {
				if (&Member).Equal(&(message.Member)) {
					*memberList = append((*memberList)[:i], (*memberList)[i+1:]...)
					fmt.Println("A member voluntarily left the group:")
					fmt.Println(message.Member)
					fmt.Println()

					logMsg, _ := json.Marshal(message.Member)
					mp2util.WriteLog("Delete: Member voluntarily left the group:" + string(logMsg))
				}
			}
		case 0: // Failure.
			for i, Member := range *memberList {
				if (&Member).Equal(&(message.Member)) {
					*memberList = append((*memberList)[:i], (*memberList)[i+1:]...)
					fmt.Println("A member was detected failed:")
					fmt.Println(message.Member)
					fmt.Println()
					// log failure info
					logMsg, _ := json.Marshal(message.Member)
					mp2util.WriteLog("Delete: Member detected as failed:" + string(logMsg))
				}
			}
		}
		// Reply to any msg except normal ping.
		if addNewMsg {
			pc.WriteTo([]byte{110, 117, 108, 108}, addr)
			// It TTL is valid, gossip later.
			if message.TTL > 0 {
				*msgQueue = append(*msgQueue, GossipMessage{Member: message.Member, Status: message.Status, TTL: message.TTL - 1})
			}
		}
	}

}

type MemberID struct {
	LocalIP    string
	JoinedTime time.Time
}

/*
	Create new ID with IP and timestamp.
*/
func NewMemberID(localIP net.IP) *MemberID {
	p := new(MemberID)
	p.LocalIP = localIP.String()
	p.JoinedTime = time.Now()
	return p
}

/*
	Helper function to detect whether itself.
*/
func (p *MemberID) Equal(comparaP *MemberID) bool {
	return p.LocalIP == comparaP.LocalIP && p.JoinedTime.Equal(comparaP.JoinedTime)
}

type GossipMessage struct {
	Member MemberID
	Status byte
	TTL    byte
}

func MemberInList(a MemberID, memberList *[]MemberID) bool {
	for _, b := range *memberList {
		if (&b).Equal(&a) {
			return true
		}
	}
	return false
}

func RemoveMember(member MemberID, memberList *[]MemberID) bool {
	for i, m := range *memberList {
		if (&m).Equal(&member) {
			*memberList = append((*memberList)[:i], (*memberList)[i+1:]...)
			return true
		}
	}
	return false
}

/*
	 When first run swim.ext, it will check whether it self is introducer.
	 If not, run JoinGroup, send a packet to the true introducer.
		The introducer will send membership in json format.
 		This node will then record this action and output message.
*/
func JoinGroup(selfMember *MemberID, msgQueue *[]GossipMessage) *[]MemberID {
	// Package the message in json format. Status 1 is join type.
	payload, _ := json.Marshal(GossipMessage{Member: *selfMember, Status: 1, TTL: 0})
	// Send it the introducer to get the memberList
	conn, err := net.Dial("udp", "172.22.158.188:12345")
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.Write(payload)
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)

	if err != nil {
		return nil
	} else {
		var memberList []MemberID
		// Deserialize the message
		err2 := json.Unmarshal(buf[:n], &memberList)

		if err2 != nil {
			log.Fatal("Unmarshal fails: ", err2)
			return nil
		} else {
			fmt.Println("Successfully join the group, start pinging the following members:")
			for _, Member := range memberList {
				fmt.Println(Member)
			}
			fmt.Println()
			// Save this log
			logMsg, _ := json.Marshal(*selfMember)
			mp2util.WriteLog("Join: Join the group as non-introducer with ID:" + string(logMsg))
			// I will later on tell other nodes that I am joined, with Gossip.
			*msgQueue = append(*msgQueue, GossipMessage{Member: *selfMember, Status: 1, TTL: 3})
			return &memberList
		}
	}
}

/*
	This node voluntarily leave the membership.
*/
func LeaveGroup(selfMember *MemberID, msgQueue *[]GossipMessage) {
	// This leave message will later be Gossiped.
	*msgQueue = append(*msgQueue, GossipMessage{Member: *selfMember, Status: 2, TTL: 3})

	fmt.Println("Leaving now and sending messages to other members.")
	logMsg, _ := json.Marshal(*selfMember)
	mp2util.WriteLog("Leave: Informing other members before leaving:" + string(logMsg))
	/*
		Wait 2 seconds before leave.
		During this period, this node will repy other pings.
			Otherwise, other nodes will think it failed before receive the leaving message.
	*/
	time.Sleep(time.Millisecond * 2000)

	logMsg, _ = json.Marshal(*selfMember)
	mp2util.WriteLog("Leave: Terminating the program:" + string(logMsg))
	fmt.Println("Thank you. See you!")

}

/*
	This is a helper function for Gossip.
	Sender will send packaged message three times with 0.02 time-out.
*/
func Gossip_sender(b []byte, member MemberID) error {
	var returnErr error
	// The message will be sent three times to promise followers can receive.
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("udp", member.LocalIP+":12345")
		if err != nil {
			log.Fatal("Gossip sending error")
		}
		defer conn.Close()

		conn.Write(b)
		// This avoids failures. If the other node doesn't reply in 0.03 second, it will temporarily be regarded as failed.
		// Once it succefully reply. Stop, I know you are alive.
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 30))
		buf := make([]byte, 1024)
		_, err2 := conn.Read(buf)

		if err2 == nil {
			return nil
		} else {
			returnErr = err2
		}
		// It will pause for 0.02 second in order to avoid a congestion.
		time.Sleep(20 * time.Millisecond)
	}
	return returnErr
}

/*
	We use a ring-based membership list. Each node will care three nodes after it.
		When getting a message, the node will first find itself position in the list, and then send the message to three followers.
		It will be applied to all kinds of messages.
*/
func Gossip(msg GossipMessage, memberList *[]MemberID, msgQueue *[]GossipMessage, selfMember *MemberID) {

	for i, Member := range *memberList {
		if (&Member).Equal(selfMember) { // Find personal positon in the list.
			var b []byte
			// Status 3 means this is just a ping message. So a simple nil will be used for efficiency.
			if msg.Status == 3 {
				buf, _ := json.Marshal(nil)
				b = buf
			} else {
				buf, _ := json.Marshal(msg)
				b = buf
			}
			// Following three others.
			for j := 0; j < 3; j++ {
				follow_id := (i + j + 1) % len(*memberList)
				recipient := (*memberList)[follow_id]

				if selfMember.Equal(&recipient) {
					continue
				}
				// This is synchroous to avoid thread-risk.
				err := Gossip_sender(b, recipient)
				// After three-time sender, if there is no reply, we think it failed.
				if err != nil {
					if RemoveMember(recipient, memberList) {
						fmt.Println("Failure detected, start informing other members, the following memeber is disconnected:")
						fmt.Println(recipient)
						fmt.Println()

						logMsg, _ := json.Marshal(recipient)
						mp2util.WriteLog("Delete: Member detected as failed:" + string(logMsg))

						*msgQueue = append(*msgQueue, GossipMessage{Member: recipient, Status: 0, TTL: 3})
					}
				}
			}
			return
		}
	}
}

/*
	This function will "clear" message in message group.
	// If there is no waiting msgs, send ping.
*/
func Pinging(memberList *[]MemberID, msgQueue *[]GossipMessage, selfMember *MemberID) {
	for {
		if len(*msgQueue) == 0 {
			var msg GossipMessage
			msg.Status = 3
			Gossip(msg, memberList, msgQueue, selfMember)
		} else {
			// fmt.Println(*msgQueue)
			Gossip((*msgQueue)[0], memberList, msgQueue, selfMember)
			*msgQueue = (*msgQueue)[1:]
		}
		time.Sleep(600 * time.Millisecond)
	}
}

/*
	Show self-memberID and membership list.
*/
func ShowList(selfMember *MemberID, memberList *[]MemberID) {
	fmt.Println("You are:")
	fmt.Println(*selfMember)

	fmt.Println("And this is your membership list:")
	for _, Member := range *memberList {
		fmt.Println(Member)
	}
	fmt.Println()
}

func main() {
	var memberList []MemberID
	var msgQueue []GossipMessage
	// Use go-routine to run multi-threads. Save incoming msgs, which will be handled in Pinging().
	go UDPListening(&memberList, &msgQueue)
	// When beginning, join the group if not introducer; Save log if introducer.
	localMember := NewMemberID(mp2util.GetLocalIP())
	if mp2util.GetLocalIP().String() != "172.22.158.188" {
		p := JoinGroup(localMember, &msgQueue)
		if p == nil {
			log.Fatal("Unable to join, because the introducer is disconnected.")
		}
		memberList = *p
	} else {
		memberList = append(memberList, *localMember)
		fmt.Println("I am the introducer, now listening to new joining members.")

		logMsg, _ := json.Marshal(*localMember)
		mp2util.WriteLog("Join: Join the group as introducer with ID:" + string(logMsg))
	}
	// Start a go-routine to handle incoming messages, pinging.
	go Pinging(&memberList, &msgQueue, localMember)

	fmt.Println("Now, you can enter commands to check your status:")
	fmt.Println("Enter 'show' to show current local memberList.")
	fmt.Println("Enter 'leave' to leave the group.")
	// Continue to read command.
	for {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		userInput := scanner.Text()

		switch userInput {
		case "show":
			ShowList(localMember, &memberList)
		case "leave":
			LeaveGroup(localMember, &msgQueue)
			return
		default:
			fmt.Println("Wrong input! Please try again:")
			fmt.Println("Enter 'show' to show current local memberList.")
			fmt.Println("Enter 'leave' to leave the group.")
		}
	}

	time.Sleep(1000 * time.Second)
}
