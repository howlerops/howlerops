package crypto_test

import (
	"fmt"
	"log"

	"github.com/jbeck018/howlerops/pkg/crypto"
)

// Example demonstrates the complete Organization Envelope Key (OEK) workflow
func Example_organizationEnvelopeKey() {
	// Step 1: Organization admin generates an OEK when creating the organization
	oek, err := crypto.GenerateOrgEnvelopeKey()
	if err != nil {
		log.Fatal(err)
	}
	defer crypto.ClearBytes(oek)
	fmt.Println("Generated OEK: [32 bytes]")

	// Step 2: Encrypt organization secrets using the OEK
	secretData := []byte("database-password-123")
	encryptedSecret, err := crypto.EncryptWithOEK(secretData, oek)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Encrypted secret has ciphertext: %t\n", encryptedSecret.Ciphertext != "")
	fmt.Printf("Encrypted secret has IV: %t\n", encryptedSecret.IV != "")
	fmt.Printf("Encrypted secret has AuthTag: %t\n", encryptedSecret.AuthTag != "")

	// Step 3: Each user gets their own encrypted copy of the OEK
	user1MasterKey, _ := crypto.GenerateMasterKey()
	defer crypto.ClearBytes(user1MasterKey)

	user2MasterKey, _ := crypto.GenerateMasterKey()
	defer crypto.ClearBytes(user2MasterKey)

	// Encrypt OEK for User 1
	oekForUser1, err := crypto.EncryptOEKForUser(oek, user1MasterKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Encrypted OEK for User 1: [stored in database]")

	// Encrypt OEK for User 2
	oekForUser2, err := crypto.EncryptOEKForUser(oek, user2MasterKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Encrypted OEK for User 2: [stored in database]")

	// Step 4: User 1 retrieves and decrypts their copy of the OEK
	user1OEK, err := crypto.DecryptOEKWithMasterKey(oekForUser1, user1MasterKey)
	if err != nil {
		log.Fatal(err)
	}
	defer crypto.ClearBytes(user1OEK)
	fmt.Println("User 1 decrypted their OEK: [32 bytes]")

	// Step 5: User 1 can now decrypt organization secrets
	decryptedSecret, err := crypto.DecryptWithOEK(encryptedSecret, user1OEK)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("User 1 decrypted secret: %s\n", string(decryptedSecret))

	// Step 6: User 2 can also access the same secret
	user2OEK, err := crypto.DecryptOEKWithMasterKey(oekForUser2, user2MasterKey)
	if err != nil {
		log.Fatal(err)
	}
	defer crypto.ClearBytes(user2OEK)

	decryptedSecret2, err := crypto.DecryptWithOEK(encryptedSecret, user2OEK)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("User 2 decrypted same secret: %s\n", string(decryptedSecret2))

	// Output:
	// Generated OEK: [32 bytes]
	// Encrypted secret has ciphertext: true
	// Encrypted secret has IV: true
	// Encrypted secret has AuthTag: true
	// Encrypted OEK for User 1: [stored in database]
	// Encrypted OEK for User 2: [stored in database]
	// User 1 decrypted their OEK: [32 bytes]
	// User 1 decrypted secret: database-password-123
	// User 2 decrypted same secret: database-password-123
}

// Example_secureKeyCleanup demonstrates proper cleanup of sensitive key material
func Example_secureKeyCleanup() {
	// Generate key
	key, err := crypto.GenerateOrgEnvelopeKey()
	if err != nil {
		log.Fatal(err)
	}

	// Always clear sensitive material when done
	defer crypto.ClearBytes(key)

	// Use the key...
	fmt.Println("Key is being used securely")

	// When function returns, defer will clear the key from memory

	// Output:
	// Key is being used securely
}
