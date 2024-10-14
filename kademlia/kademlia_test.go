package kademlia

import (
	"testing"
)

func TestLookupContact(t *testing.T) {
	// Sample contacts
	/*var details []Detail = []Detail{
		GetContactDetails("ffffffff00000000000000000000000000000000", "localhost:8000"),
		GetContactDetails("1111111100000000000000000000000000000000", "localhost:8001"),
		GetContactDetails("1111111200000000000000000000000000000000", "localhost:8002"),
		GetContactDetails("1111111300000000000000000000000000000000", "localhost:8003"),
	}

	var quantity int = len(details)
	var contacts []Contact = []Contact{}
	for _, detail := range details {
		contacts = append(contacts, GetContact(detail))
	}

	var k Kademlia = *NewKademlia(contacts[0])
	for _, contact := range contacts {
		k.Rt.AddContact(contact)
	}
	var response []Contact = k.LookupContact(*contacts[1].ID)
	if len(response) != quantity {
		t.Error("[FAIL] Incorrect closest contacts returned")
	}*/
}

func TestUpdateContact(t *testing.T) {
	var target KademliaID
	var closest ContactCandidates
	var contacted map[string]bool = map[string]bool{}
	var contacts []Contact
	responses := make(chan Message, 5)

	var me = NewContact(NewKademliaID("FFFFFFFF00000000000000000000000000000000"), "127.0.0.1:1234")

	var k Kademlia = *NewKademlia(me)

	k.updateContacts(&contacted, &closest, &contacts, responses, target, k.Network.SendFindContactMessage)
}
