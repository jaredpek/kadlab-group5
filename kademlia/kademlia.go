package kademlia

import (
	"crypto/sha1"
	"fmt"
	"log"
	"strconv"
	"time"
)

const tRepublish = 1 * time.Minute

const Alpha = 3

// Default network values
const BootstrapIP = "172.26.0.2:1234"
const ListenPort = "1234"
const PacketSize = 1024 * 4

type Kademlia struct {
	Network *Network
	Rt      *RoutingTable
}

// Creates a new instance of the Kademlia
func NewKademlia(me Contact) *Kademlia {
	Rt := NewRoutingTable(me)
	return &Kademlia{
		Network: &Network{
			Rt:                Rt,
			BootstrapIP:       BootstrapIP,
			ListenPort:        ListenPort,
			PacketSize:        PacketSize,
			ExpectedResponses: make(map[KademliaID]chan Message, 10),
			Messenger:         &UDPMessenger{Rt: Rt},
		},
		Rt: Rt,
	}
}

// Local function used to update what contacts have been contacted
func (kademlia *Kademlia) updateContacts(
	contacted *map[string]bool,
	closest *ContactCandidates,
	contacts *[]Contact,
	responses chan Message,
	target KademliaID,
	findFunc func(KademliaID, *Contact, chan Message),
) {
	// For each contact of the k-closest
	for _, closestContact := range closest.GetContacts(bucketSize) {
		// Continue to the next contact if already contacted
		if (*contacted)[closestContact.Address] {
			continue
		}

		// Stop sending find contact requests if reached alpha nodoes
		if len(*contacts) >= Alpha {
			break
		}

		// Send lookup request to the node async
		go findFunc(target, &closestContact, responses)

		// Update contact record status
		(*contacted)[closestContact.Address] = true
		*contacts = append(*contacts, closestContact)
	}
}

// Implements NodeLookup in Kademlia. Finds the k closest nodes to an KademlaiID.
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
		kademlia.updateContacts(&contacted, &closest, &contacts, responses, target, kademlia.Network.SendFindContactMessage)

		// For each contact that was sent a find contact message
		for i := 0; i < len(contacts); i++ {
			// Receive the response from the channel
			message := <-responses

			// Print list of contacts and add contact to routing table
			log.Println("[FIND_CONTACT] Got contact response: ")
			for _, foundContact := range message.Contacts {
				sender := foundContact
				sender.CalcDistance(kademlia.Rt.me.ID) // calc distance to self
				fmt.Println("SENDER ID:", sender.ID, "\nME ID:", kademlia.Rt.me.ID, "\nDISTANCE:", sender.distance, "\nDISTANCE ADDRESS", &sender.distance)
				fmt.Println("CURRENT SENDER:", sender)
				kademlia.Rt.AddContact(sender, kademlia.Network.SendPingMessage)
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

// Used when a node joins a kademlia network.
func (kademlia *Kademlia) JoinNetwork() {
	for {
		log.Println("Joining network")
		response := make(chan Message)
		go kademlia.Network.SendPingMessage(&Contact{Address: kademlia.Network.BootstrapIP}, response) // ping bootstrap node so that it is added to routing table
		r := <-response
		if r.MsgType == "PONG" {
			break
		} else {
			log.Println("Timeout joining network")
		}
	}

	kademlia.LookupContact(*kademlia.Rt.me.ID) // lookup on this node to add close nodes to routing table
	rtInfo := "[JOIN] Routing table after joining:\n"
	currRt := kademlia.Network.Rt.buckets

	for i, val := range currRt {
		rtInfo += "Content in bucket " + strconv.Itoa(i) + "\n"
		for e := val.list.Front(); e != nil; e = e.Next() {
			rtInfo += "  " + e.Value.(Contact).ID.String() + "\n"
		}
	}

	fmt.Println(rtInfo)

}

// should return a string with the result. if the data could be found a string with the data and node it
// was retrived from should be returned. otherwise just return that the file could not be found
func (kademlia *Kademlia) LookupData(hash string) string {
	log.Println("[FIND_DATA] Performing lookup data")
	var closest ContactCandidates
	var contacted map[string]bool = map[string]bool{}
	responses := make(chan Message, 5)
	id := NewKademliaID(hash)

	// For each contact of the initial k-closest contacts to the target
	for _, contact := range kademlia.Rt.FindClosestContacts(id, bucketSize) {
		// Calculate the distance to the target
		contact.CalcDistance(id)

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
		kademlia.updateContacts(&contacted, &closest, &contacts, responses, *id, kademlia.Network.SendFindDataMessage)

		// For each contact that was sent a find contact message
		for i := 0; i < len(contacts); i++ {
			// Receive the response from the channel
			message := <-responses
			if message.Body != "" {
				return message.Body
			}

			// Add contacts to routing table
			for _, foundContact := range message.Contacts {
				sender := foundContact
				sender.CalcDistance(kademlia.Rt.me.ID) // calc distance to self
				go kademlia.Rt.AddContact(sender, kademlia.Network.SendPingMessage)
			}

			// For each contact that was received from the message
			for j := 0; j < len(message.Contacts); j++ {
				// Calculate the distance between the target and the contact
				message.Contacts[j].CalcDistance(id)
			}

			// Add the found contacts to the list of closest contacts
			closest.Append(message.Contacts)
		}

		// If there are no k closest contacts that are uncontacted, return k closest contacts
		if len(contacts) == 0 {
			return "The requested object could not be downloaded"
		}
	}
}

// should return the hash of the data if it was successfully uploaded.
// an error should be returned if the data could not be uploaded
func (kademlia *Kademlia) Store(data []byte) (error, string) {
	// check that data fits requriements for KademliaID
	h := sha1.New()
	h.Write(data)
	var res [IDLength]byte
	copy(res[:], h.Sum(nil))

	var dataID KademliaID = res

	// find the K nearest nodes
	closestNodes := kademlia.LookupContact(dataID)

	for _, n := range closestNodes {
		fmt.Println("[STORE]: Closest nodes to string:", "\n ID:", n.ID, "\n ADDRESS", n.Address, "\n DIST:", n.distance)
	}

	// send Store instruction to each node
	for _, n := range closestNodes {
		kademlia.Network.SendStoreMessage(dataID, data, &n)
	}

	go func() { // republish the data
		time.Sleep(tRepublish)
		kademlia.Store(data)
	}()

	return nil, dataID.String()
}
