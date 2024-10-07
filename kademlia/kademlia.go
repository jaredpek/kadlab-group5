package kademlia

import (
	"fmt"
	"log"
	"strconv"
)

const Alpha = 3

// Default network values
const BootstrapIP = "172.26.0.2:1234"
const ListenPort = "1234"
const PacketSize = 1024

type Kademlia struct {
	Network *Network
	Rt      *RoutingTable
}

func NewKademlia(me Contact) *Kademlia {
	Rt := NewRoutingTable(me)
	return &Kademlia{
		Network: &Network{
			Rt:                Rt,
			BootstrapIP:       BootstrapIP,
			ListenPort:        ListenPort,
			PacketSize:        PacketSize,
			ExpectedResponses: make(map[KademliaID]chan Message, 10),
		},
		Rt: Rt,
	}
}

func (kademlia *Kademlia) LookupContact(target KademliaID) []Contact {
	log.Println("[FIND_CONTACT] Performing lookup contact")
	var closest ContactCandidates
	var contacted map[string]bool = map[string]bool{}
	responses := make(chan Message, 5)

	// For each contact of the initial k-closest contacts to the target
	for _, contact := range kademlia.Rt.FindClosestContacts(&target, bucketSize) {
		// Calculate the distance to the target
		contact.CalcDistance(&target)

		// Create record for new contact
		contacted[contact.Address] = false

		// Add it to the closest list
		closest.Append([]Contact{contact})
	}

	for {
		var contacts []Contact

		// Sort the contacts by their distance
		closest.Sort()

		// For each contact of the k-closest
		for _, closestContact := range closest.GetContacts(bucketSize) {
			// Continue to the next contact if already contacted
			if contacted[closestContact.Address] {
				continue
			}

			// Stop sending find contact requests if reached alpha nodoes
			if len(contacts) >= Alpha {
				break
			}

			// Send node lookup request to the node async
			go kademlia.Network.SendFindContactMessage(target, &closestContact, responses)

			// Update contact record status
			contacted[closestContact.Address] = true
			contacts = append(contacts, closestContact)
		}

		// For each contact that was sent a find contact message
		for i := 0; i < len(contacts); i++ {
			// Receive the response from the channel
			message := <-responses

			// Print list of contacts
			log.Println("[FIND_CONTACT] Got contact response: ")
			for _, foundContact := range message.Contacts {
				kademlia.Network.Rt.lock.Lock()
				sender := foundContact
				sender.CalcDistance(kademlia.Rt.me.ID) // calc distance to self
				kademlia.Network.Rt.AddContact(sender)
				kademlia.Network.Rt.lock.Unlock()
				log.Printf("  %s\n", foundContact.ID.String())
			}

			// For each contact that was received from the message
			for j := 0; j < len(message.Contacts); j++ {
				// Calculate the distance between the target and the contact
				message.Contacts[j].CalcDistance(&target)
			}

			// Add the found contacts to the list of closest contacts
			closest.Append(message.Contacts)
		}

		// If there are no k closest contacts that are uncontacted, return k closest contacts
		if len(contacts) == 0 {
			return closest.GetContacts(bucketSize)
		}
	}
}

func (kademlia *Kademlia) JoinNetwork() {
	log.Println("Joining network")
	response := make(chan Message)
	go kademlia.Network.SendPingMessage(&Contact{Address: kademlia.Network.BootstrapIP}, response) // ping bootstrap node so that it is added to routing table
	r := <-response
	log.Println(r.MsgType)                     // wait for response
	kademlia.LookupContact(*kademlia.Rt.me.ID) // lookup on this node to add close nodes to routing table

	rtInfo := "Routing table:\n"
	currRt := kademlia.Network.Rt.buckets

	for i, val := range currRt {
		rtInfo += "Content in bucket " + strconv.Itoa(i) + "\n"
		for e := val.list.Front(); e != nil; e = e.Next() {
			rtInfo += "  " + e.Value.(Contact).ID.String() + "\n"
		}
	}

	fmt.Println(rtInfo)

	fmt.Println("my id:", kademlia.Rt.me.ID)
}

// should return a string with the result. if the data could be found a string with the data and node it
// was retrived from should be returned. otherwise just return that the file could not be found
func (kademlia *Kademlia) LookupData(hash string) string {
	// TODO
	// return "The requested file could not be downloaded"
	panic("LookupData not implemented")
}

// should return the hash of the data if it was successfully uploaded.
// an error should be returned if the data could not be uploaded
func (kademlia *Kademlia) Store(data []byte) (error, string) {
	// TODO
	panic("Store not implemented")
}
